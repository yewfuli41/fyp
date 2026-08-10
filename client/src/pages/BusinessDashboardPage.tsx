import { useCallback, useEffect, useMemo, useState } from "react";
import { Alert, Button, ButtonGroup, Col, Container, Form, Row, Spinner } from "react-bootstrap";
import { useAuth } from "../auth/AuthContext";
import { addDays, toISO, todayISO } from "../utils/serviceSlotHelpers";
import {
    getBookingSummary, getBookingTrend, getServicePopularity, getSlotUtilization,
    getStaffUtilization, getCustomerRetention, getCancellationAnalysis, getCancellationServiceOptions,
    getBusinessCreatedAt,
    type BookingSummary, type BookingTrendPoint, type ServicePopularity, type SlotUtilization,
    type SlotUtilizationGroupBy, type StaffUtilization, type CustomerRetention, type CancellationAnalysis,
    type TrendGranularity, type ServiceFilterOption,
} from "../services/AnalyticsService";
import { getBusinessStaff, type Staff } from "../services/StaffService";
import DashboardBookingSummary from "../components/DashboardBookingSummary";
import DashboardBookingTrend from "../components/DashboardBookingTrend";
import DashboardServicePopularity from "../components/DashboardServicePopularity";
import DashboardSlotUtilization from "../components/DashboardSlotUtilization";
import DashboardStaffUtilization from "../components/DashboardStaffUtilization";
import DashboardCustomerRetention from "../components/DashboardCustomerRetention";
import DashboardCancellationAnalysis from "../components/DashboardCancellationAnalysis";
import "../styles/BusinessDashboard.css";

type FilterMode = "years" | "months" | "dates";

const rangeForDays = (days: number): { from: string; until: string } => {
    const today = new Date();
    return { from: toISO(addDays(today, -(days - 1))), until: todayISO() };
};

// A future end clamps to today — the year/month itself may not be over yet.
const clampUntil = (until: string): string => (until > todayISO() ? todayISO() : until);

const rangeForYearSpan = (fromYear: number, toYear: number): { from: string; until: string } => ({
    from: toISO(new Date(fromYear, 0, 1)),
    until: clampUntil(toISO(new Date(toYear, 11, 31))),
});

// fromMonth/toMonth are "YYYY-MM", the native <input type="month"> format.
const rangeForMonthSpan = (fromMonth: string, toMonth: string): { from: string; until: string } => {
    const [fy, fm] = fromMonth.split("-").map(Number);
    const [ty, tm] = toMonth.split("-").map(Number);
    return {
        from: toISO(new Date(fy, fm - 1, 1)),
        until: clampUntil(toISO(new Date(ty, tm, 0))), // day 0 of next month = last day of `tm`
    };
};

// Booking Trend gets noisy at daily resolution over a long range — step up
// the bucket size as the selected span grows.
const granularityForSpan = (from: string, until: string): TrendGranularity => {
    const days = Math.round((new Date(`${until}T00:00:00`).getTime() - new Date(`${from}T00:00:00`).getTime()) / 86_400_000);
    if (days <= 31) return "day";
    if (days <= 180) return "week";
    return "month";
};

const CURRENT_YEAR = new Date().getFullYear();
const THIS_MONTH = todayISO().slice(0, 7);

// [businessCreatedYear, currentYear] — years the business couldn't possibly
// have data outside of. Falls back to just the current year while the
// business's creation date hasn't loaded yet.
const yearOptionsFrom = (businessCreatedYear: number | null): number[] => {
    const startYear = businessCreatedYear ?? CURRENT_YEAR;
    const length = Math.max(1, CURRENT_YEAR - startYear + 1);
    return Array.from({ length }, (_, i) => startYear + i);
};

const formatRangeLabel = (from: string, until: string): string => {
    const fmt = (iso: string) => new Date(`${iso}T00:00:00`).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
    return `${fmt(from)} – ${fmt(until)}`;
};

interface DashboardData {
    summary: BookingSummary | null;
    trend: BookingTrendPoint[];
    popularity: ServicePopularity[];
    staffUtilization: StaffUtilization[];
    retention: CustomerRetention | null;
}

const EMPTY_DATA: DashboardData = {
    summary: null, trend: [], popularity: [], staffUtilization: [], retention: null,
};

// The business owner's landing page (UC-12) — each card below fires its own
// GraphQL query (see AnalyticsService.ts), so nothing is overfetched.
export default function BusinessDashboardPage() {
    const { token } = useAuth();
    const activeToken = token ?? localStorage.getItem("token");

    const [filterMode, setFilterMode] = useState<FilterMode>("dates");
    const [yearFrom, setYearFrom] = useState(CURRENT_YEAR);
    const [yearTo, setYearTo] = useState(CURRENT_YEAR);
    const [monthFrom, setMonthFrom] = useState(THIS_MONTH);
    const [monthTo, setMonthTo] = useState(THIS_MONTH);
    const [range, setRange] = useState(() => rangeForDays(30));
    const [data, setData] = useState<DashboardData>(EMPTY_DATA);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState("");

    // Cancellation Analysis gets its own staff/service filter, so it's
    // fetched separately from the rest of the dashboard — changing the
    // filter shouldn't refetch every other card.
    const [staffList, setStaffList] = useState<Staff[]>([]);
    const [serviceOptions, setServiceOptions] = useState<ServiceFilterOption[]>([]);
    const [cancelStaffIds, setCancelStaffIds] = useState<string[]>([]);
    // Selected service *names* — expanded to serviceIds (below) right before
    // querying, since one name can span multiple ids (see ServiceFilterOption).
    // Memoized so the expanded array is referentially stable across renders
    // that don't change the selection, keeping it safe as a useCallback dep.
    const [cancelServiceNames, setCancelServiceNames] = useState<string[]>([]);
    const cancelServiceIds = useMemo(
        () => serviceOptions.filter(o => cancelServiceNames.includes(o.serviceName)).flatMap(o => o.serviceIds),
        [serviceOptions, cancelServiceNames],
    );
    const [cancellation, setCancellation] = useState<CancellationAnalysis | null>(null);
    const [isCancellationLoading, setIsCancellationLoading] = useState(true);
    const [cancellationError, setCancellationError] = useState("");

    // Bounds every filter picker below — null until loaded, meaning "no
    // lower bound yet" in the meantime.
    const [businessCreatedAt, setBusinessCreatedAt] = useState<string | null>(null);
    const businessCreatedYear = useMemo(
        () => (businessCreatedAt ? new Date(`${businessCreatedAt}T00:00:00`).getFullYear() : null),
        [businessCreatedAt],
    );
    const businessCreatedMonth = businessCreatedAt?.slice(0, 7); // "YYYY-MM"
    const yearOptions = useMemo(() => yearOptionsFrom(businessCreatedYear), [businessCreatedYear]);

    useEffect(() => {
        if (!activeToken) return;
        getBusinessStaff(activeToken).then(res => setStaffList(res.data?.displayStaff ?? [])).catch(() => setStaffList([]));
        // Includes soft-deleted services — a deleted service's past
        // cancellations/rejections still appear in the byService breakdown.
        getCancellationServiceOptions(activeToken)
            .then(res => setServiceOptions(res.data?.cancellationServiceOptions ?? []))
            .catch(() => setServiceOptions([]));
        getBusinessCreatedAt(activeToken)
            .then(res => {
                const createdAt = res.data?.businessCreatedAt;
                if (!createdAt) return;
                setBusinessCreatedAt(createdAt);
                // The initial "last 30 days" default (picked before we knew
                // the business's age) may reach earlier than it existed —
                // clamp it now that we know.
                setRange(r => (r.from < createdAt ? { from: createdAt, until: r.until } : r));
            })
            .catch(() => setBusinessCreatedAt(null));
    }, [activeToken]);

    const fetchCancellation = useCallback(async () => {
        if (!activeToken) return;
        setIsCancellationLoading(true);
        setCancellationError("");
        try {
            const res = await getCancellationAnalysis(
                activeToken, range.from, range.until, cancelStaffIds, cancelServiceIds,
            );
            if (res.errors?.length) {
                setCancellationError(res.errors[0].message);
                setCancellation(null);
                return;
            }
            setCancellation(res.data?.cancellationAnalysis ?? null);
        } catch {
            setCancellationError("Failed to load cancellation analysis.");
            setCancellation(null);
        } finally {
            setIsCancellationLoading(false);
        }
    }, [activeToken, range, cancelStaffIds, cancelServiceIds]);

    useEffect(() => { fetchCancellation(); }, [fetchCancellation]);

    // Slot Utilization gets its own weekday/date/month/year breakdown toggle,
    // so it's fetched separately from the rest of the dashboard — changing
    // it shouldn't refetch every other card.
    const [slotGroupBy, setSlotGroupBy] = useState<SlotUtilizationGroupBy>("weekday");
    const [slotUtilization, setSlotUtilization] = useState<SlotUtilization | null>(null);
    const [isSlotLoading, setIsSlotLoading] = useState(true);
    const [slotError, setSlotError] = useState("");

    const fetchSlotUtilization = useCallback(async () => {
        if (!activeToken) return;
        setIsSlotLoading(true);
        setSlotError("");
        try {
            const res = await getSlotUtilization(activeToken, range.from, range.until, slotGroupBy);
            if (res.errors?.length) {
                setSlotError(res.errors[0].message);
                setSlotUtilization(null);
                return;
            }
            setSlotUtilization(res.data?.slotUtilization ?? null);
        } catch {
            setSlotError("Failed to load slot utilization.");
            setSlotUtilization(null);
        } finally {
            setIsSlotLoading(false);
        }
    }, [activeToken, range, slotGroupBy]);

    useEffect(() => { fetchSlotUtilization(); }, [fetchSlotUtilization]);

    const fetchAll = useCallback(async () => {
        if (!activeToken) return;
        setIsLoading(true);
        setError("");
        try {
            const granularity = granularityForSpan(range.from, range.until);
            const [summaryRes, trendRes, popularityRes, staffRes, retentionRes] = await Promise.all([
                getBookingSummary(activeToken, range.from, range.until),
                getBookingTrend(activeToken, range.from, range.until, granularity),
                getServicePopularity(activeToken, range.from, range.until),
                getStaffUtilization(activeToken, range.from, range.until),
                getCustomerRetention(activeToken, range.from, range.until),
            ]);
            const firstError = [summaryRes, trendRes, popularityRes, staffRes, retentionRes]
                .find(r => r.errors?.length)?.errors?.[0];
            if (firstError) {
                setError(firstError.message);
                return;
            }
            setData({
                summary: summaryRes.data?.bookingSummary ?? null,
                trend: trendRes.data?.bookingTrend ?? [],
                popularity: popularityRes.data?.servicePopularity ?? [],
                staffUtilization: staffRes.data?.staffUtilization ?? [],
                retention: retentionRes.data?.customerRetention ?? null,
            });
        } catch {
            setError("Failed to load analytics.");
        } finally {
            setIsLoading(false);
        }
    }, [activeToken, range]);

    useEffect(() => { fetchAll(); }, [fetchAll]);

    // Each apply* funnels its picker(s) through here so choosing either end
    // applies immediately, swapped if the user picked them backwards.
    const applyYears = (fromYear: number, toYear: number) => {
        setYearFrom(fromYear);
        setYearTo(toYear);
        setFilterMode("years");
        const [lo, hi] = fromYear <= toYear ? [fromYear, toYear] : [toYear, fromYear];
        setRange(rangeForYearSpan(lo, hi));
    };

    const applyMonths = (fromMonth: string, toMonth: string) => {
        setMonthFrom(fromMonth);
        setMonthTo(toMonth);
        setFilterMode("months");
        const [lo, hi] = fromMonth <= toMonth ? [fromMonth, toMonth] : [toMonth, fromMonth];
        setRange(rangeForMonthSpan(lo, hi));
    };

    const applyDates = (from: string, until: string) => {
        if (!from || !until) return;
        setFilterMode("dates");
        setRange(from <= until ? { from, until } : { from: until, until: from });
    };

    return (
        <Container fluid className="py-4 px-4 dashboard-page">
            <div className="d-flex flex-wrap justify-content-between align-items-center mb-4 gap-3">
                <h1 className="fs-2 fw-bold mb-0">Dashboard</h1>

                <div className="dashboard-filter-bar d-flex flex-wrap align-items-center gap-3">
                    <ButtonGroup size="sm">
                        {([["years", "Years"], ["months", "Months"], ["dates", "Dates"]] as [FilterMode, string][]).map(([key, label]) => (
                            <Button
                                key={key}
                                variant={filterMode === key ? "primary" : "outline-secondary"}
                                onClick={() => {
                                    if (key === "years") applyYears(yearFrom, yearTo);
                                    else if (key === "months") applyMonths(monthFrom, monthTo);
                                    else setFilterMode("dates");
                                }}
                            >
                                {label}
                            </Button>
                        ))}
                    </ButtonGroup>

                    <div className="vr d-none d-sm-block" />

                    {filterMode === "years" && (
                        <div className="d-flex align-items-center gap-2">
                            <Form.Select
                                size="sm"
                                className="dashboard-filter-input"
                                value={yearFrom}
                                onChange={e => applyYears(Number(e.target.value), yearTo)}
                            >
                                {yearOptions.map(y => <option key={y} value={y}>{y}</option>)}
                            </Form.Select>
                            <span className="text-muted">–</span>
                            <Form.Select
                                size="sm"
                                className="dashboard-filter-input"
                                value={yearTo}
                                onChange={e => applyYears(yearFrom, Number(e.target.value))}
                            >
                                {yearOptions.map(y => <option key={y} value={y}>{y}</option>)}
                            </Form.Select>
                        </div>
                    )}
                    {filterMode === "months" && (
                        <div className="d-flex align-items-center gap-2">
                            <Form.Control
                                type="month"
                                size="sm"
                                className="dashboard-filter-input"
                                min={businessCreatedMonth}
                                max={monthTo}
                                value={monthFrom}
                                onChange={e => applyMonths(e.target.value, monthTo)}
                            />
                            <span className="text-muted">–</span>
                            <Form.Control
                                type="month"
                                size="sm"
                                className="dashboard-filter-input"
                                min={monthFrom}
                                max={THIS_MONTH}
                                value={monthTo}
                                onChange={e => applyMonths(monthFrom, e.target.value)}
                            />
                        </div>
                    )}
                    {filterMode === "dates" && (
                        <div className="d-flex align-items-center gap-2">
                            <Form.Control
                                type="date"
                                size="sm"
                                className="dashboard-filter-input"
                                min={businessCreatedAt ?? undefined}
                                max={range.until}
                                value={range.from}
                                onChange={e => applyDates(e.target.value, range.until)}
                            />
                            <span className="text-muted">–</span>
                            <Form.Control
                                type="date"
                                size="sm"
                                className="dashboard-filter-input"
                                min={range.from}
                                max={todayISO()}
                                value={range.until}
                                onChange={e => applyDates(range.from, e.target.value)}
                            />
                        </div>
                    )}

                    <span className="text-muted small fw-semibold">{formatRangeLabel(range.from, range.until)}</span>
                </div>
            </div>

            {error && <Alert variant="danger">{error}</Alert>}

            {isLoading ? (
                <div className="text-center py-5"><Spinner animation="border" variant="primary" /></div>
            ) : (
                <Row className="g-4">
                    <Col xs={12}>
                        <DashboardBookingSummary summary={data.summary} />
                    </Col>
                    <Col xs={12} lg={7}>
                        <DashboardBookingTrend points={data.trend} />
                    </Col>
                    <Col xs={12} lg={5}>
                        <DashboardServicePopularity items={data.popularity} />
                    </Col>
                    <Col xs={12} lg={7}>
                        <DashboardSlotUtilization
                            utilization={slotUtilization}
                            isLoading={isSlotLoading}
                            error={slotError}
                            groupBy={slotGroupBy}
                            onGroupByChange={setSlotGroupBy}
                        />
                    </Col>
                    <Col xs={12} lg={5}>
                        <DashboardStaffUtilization items={data.staffUtilization} />
                    </Col>
                    <Col xs={12} lg={5}>
                        <DashboardCustomerRetention retention={data.retention} />
                    </Col>
                    <Col xs={12} lg={7}>
                        <DashboardCancellationAnalysis
                            analysis={cancellation}
                            isLoading={isCancellationLoading}
                            error={cancellationError}
                            staffList={staffList}
                            serviceList={serviceOptions}
                            staffIds={cancelStaffIds}
                            serviceNames={cancelServiceNames}
                            onStaffChange={setCancelStaffIds}
                            onServiceNamesChange={setCancelServiceNames}
                        />
                    </Col>
                </Row>
            )}
        </Container>
    );
}
