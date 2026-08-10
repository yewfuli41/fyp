import { Outlet, Navigate, Route, Routes } from "react-router-dom";
import LogInPage from "./pages/LogInPage";
import SignUpPage from "./pages/SignUpPage";
import ProfilePage from "./pages/ProfilePage";
import EditBusinessPage from "./pages/EditBusinessPage";
import RegisterBusinessPage from "./pages/RegisterBusinessPage";
import ServicePage from "./pages/ServicePage";
import RegisterStaffPage from "./pages/RegisterStaffPage"
import StaffManagementPage from "./pages/StaffManagementPage"
import StaffAvailabilityPage from "./pages/StaffAvailabilityPage"
import LeaveApplicationPage from "./pages/LeaveApplicationPage"
import CalendarPage from "./pages/CalendarPage"
import ResetPasswordPage from "./pages/ResetPasswordPage";
import ProtectedRoute from "./components/ProtectedRoute";
import BusinessListPage from "./pages/BusinessListPage";
import BusinessServicesPage from "./pages/BusinessServicesPage";
import BookLandingPage from "./pages/BookLandingPage";
import AppointmentsPage from "./pages/AppointmentsPage";
import AppNavbar from "./components/AppNavbar";
import HomePage from "./pages/HomePage";

function Layout() {
  return (
    <>
      <AppNavbar />
      <Outlet />
    </>
  );
}

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<LogInPage />} />
        <Route path="/signup" element={<SignUpPage />} />
        <Route path="/book" element={<BookLandingPage />} />
        <Route path="/businesses" element={<BusinessListPage />} />
        <Route path="/businesses/:businessId/services" element={<BusinessServicesPage />} />
        {/* Not behind ProtectedRoute — an anonymous visitor still lands here
            (from the home page's "Register Business" button) and sees a
            sign-up prompt instead of a generic /login bounce. */}
        <Route path="/register-business" element={<RegisterBusinessPage />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="/appointments" element={<AppointmentsPage />} />
        </Route>
        <Route element={<ProtectedRoute roles={["STAFF"]} />}>
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/my-calendar" element={<CalendarPage />} />
          <Route path="/leave" element={<LeaveApplicationPage />} />
        </Route>
        <Route element={<ProtectedRoute roles={["OWNER"]} />}>
          <Route path="/edit-business" element={<EditBusinessPage />} />
          <Route path="/services" element={<ServicePage />} />
          <Route path="/register-staff" element={<RegisterStaffPage />} />
          <Route path="/staff" element={<StaffManagementPage />} />
          <Route path="/staff-availability" element={<StaffAvailabilityPage />} />
          <Route path="/service-slots" element={<CalendarPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/book" replace />} />
      </Route>
    </Routes>
  );
}

export default App;
