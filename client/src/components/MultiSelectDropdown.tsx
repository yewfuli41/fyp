import { useState } from "react";
import { Badge, Dropdown, Form } from "react-bootstrap";

interface Option {
    value: string;
    label: string;
}

interface Props {
    label: string;
    options: Option[];
    selected: string[];
    onChange: (values: string[]) => void;
}

// A compact, searchable multi-select — one button that opens a popover with a
// search box and a scrollable checkable list. Scales to long lists (many staff
// or services) where a row of pills/checkboxes would overflow the toolbar.
export default function MultiSelectDropdown({ label, options, selected, onChange }: Props) {
    const [query, setQuery] = useState("");
    const q = query.trim().toLowerCase();
    const filtered = q ? options.filter(o => o.label.toLowerCase().includes(q)) : options;

    const toggle = (v: string) =>
        onChange(selected.includes(v) ? selected.filter(x => x !== v) : [...selected, v]);

    return (
        <Dropdown autoClose="outside">
            <Dropdown.Toggle variant="outline-secondary" size="sm"  style={{ minWidth: 130 }}>
                {label}
                {selected.length > 0 && <Badge bg="primary" pill className="ms-2">{selected.length}</Badge>}
            </Dropdown.Toggle>
            <Dropdown.Menu style={{ minWidth: 240 }} className="p-2">
                <Form.Control
                    size="sm"
                    placeholder={`Search ${label.toLowerCase()}…`}
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                    className="mb-2"
                    autoFocus
                />
                <div style={{ maxHeight: 240, overflowY: "auto" }}>
                    {filtered.length === 0 ? (
                        <div className="text-muted small px-1 py-1">No matches</div>
                    ) : filtered.map(o => {
                        const isSelected = selected.includes(o.value);
                        return (
                            <div
                                key={o.value}
                                className={`rounded px-2 py-1${isSelected ? " fw-semibold" : ""}`}
                                style={isSelected ? { background: "#e7f1ff" } : undefined}
                            >
                                <Form.Check
                                    type="checkbox"
                                    id={`msf-${label}-${o.value}`}
                                    label={o.label}
                                    checked={isSelected}
                                    onChange={() => toggle(o.value)}
                                />
                            </div>
                        );
                    })}
                </div>
                {selected.length > 0 && (
                    <div className="d-flex justify-content-between align-items-center mt-2 pt-2 border-top">
                        <span className="text-muted small">{selected.length} selected</span>
                        <button type="button" className="btn btn-link btn-sm p-0" onClick={() => onChange([])}>
                            Clear
                        </button>
                    </div>
                )}
            </Dropdown.Menu>
        </Dropdown>
    );
}
