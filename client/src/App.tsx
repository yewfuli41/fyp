import { Outlet, Navigate, Route, Routes } from "react-router-dom";
import LogInPage from "./pages/LogInPage";
import SignUpPage from "./pages/SignUpPage";
import ProfilePage from "./pages/ProfilePage";
import EditBusinessPage from "./pages/EditBusinessPage";
import RegisterBusinessPage from "./pages/RegisterBusinessPage";
import ServicePage from "./pages/ServicePage";
import RegisterStaffPage from "./pages/RegisterStaffPage"
import StaffManagementPage from "./pages/StaffManagementPage"
import CalendarPage from "./pages/CalendarPage"
import ResetPasswordPage from "./pages/ResetPasswordPage";
import ProtectedRoute from "./components/ProtectedRoute";
import BusinessListPage from "./pages/BusinessListPage";
import BookLandingPage from "./pages/BookLandingPage";
import BusinessServicesPage from "./pages/BusinessServicesPage";
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
        <Route element={<ProtectedRoute />}>
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="/register-business" element={<RegisterBusinessPage />} />
        </Route>
        <Route element={<ProtectedRoute roles={["STAFF"]} />}>
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/my-calendar" element={<CalendarPage />} />
        </Route>
        <Route element={<ProtectedRoute roles={["OWNER"]} />}>
          <Route path="/edit-business" element={<EditBusinessPage />} />
          <Route path="/services" element={<ServicePage />} />
          <Route path="/register-staff" element={<RegisterStaffPage />} />
          <Route path="/staff" element={<StaffManagementPage />} />
          <Route path="/service-slots" element={<CalendarPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

export default App;
