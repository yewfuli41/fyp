import { useState, useEffect } from "react";
import type { FormEvent } from "react";
import { Alert, Button, Container, Form, Spinner } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { userProfile, updateProfile } from "../services/ProfileService";
import { applyGraphQLErrors, type FieldErrors } from "../utils/graphqlErrors";
import { FIELD_LIMITS } from "../utils/fieldLimits";
import userIcon from "../assets/user-icon-simple-design-free-vector.jpg";
import emailIcon from "../assets/message-icon-logo-design-vector.webp";
import phoneIcon from "../assets/phone--v1.jpg";
import "../styles/ProfilePage.css";

export default function ProfilePage() {
  const navigate = useNavigate();
  const { token, login } = useAuth();
  const activeToken = token ?? localStorage.getItem("token");

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [contactNumber, setContactNumber] = useState("");
  const [savedProfile, setSavedProfile] = useState({
    username: "",
    email: "",
    contactNumber: "",
  });

  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [formSuccess, setFormSuccess] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isEditing, setIsEditing] = useState(false);
  const [businessProfile, setBusinessProfile] = useState<any>(null);

  useEffect(() => {
    if (!activeToken) return;

    const fetchProfile = async () => {
      try {
        const result = await userProfile(activeToken);
        if (result.data?.userProfile) {
          const profile = result.data.userProfile;
          setUsername(profile.username);
          setEmail(profile.email);
          setContactNumber(profile.contactNumber || "");
          setSavedProfile({
            username: profile.username,
            email: profile.email,
            contactNumber: profile.contactNumber || "",
          });
          setBusinessProfile(profile.businessProfile);
        } else if (result.errors?.length) {
          setFormError(result.errors[0].message);
        }
      } catch {
        setFormError("Failed to load profile data.");
      } finally {
        setIsLoading(false);
      }
    };

    fetchProfile();
  }, [activeToken, navigate]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!activeToken) return;

    setFieldErrors({});
    setFormError("");
    setFormSuccess("");

    setIsSubmitting(true);

    try {
      const result = await updateProfile(activeToken, username, email, contactNumber);

      if (applyGraphQLErrors(result, {
        setFieldErrors,
        setFormError,
        fallbackMessage: "Profile update failed",
      })) return;

      const updatedUser = result.data?.updateProfile;
      if (!updatedUser) {
        setFormError("Profile update failed");
        return;
      }

      setFormSuccess("Profile updated successfully!");
      setIsEditing(false);
      setSavedProfile({
        username: updatedUser.username,
        email: updatedUser.email,
        contactNumber: updatedUser.contactNumber || "",
      });

      // Update the auth context with new user data
      login(activeToken, {
        userId: Number(updatedUser.userId),
        username: updatedUser.username,
        email: updatedUser.email,
        contactNumber: updatedUser.contactNumber,
      });
    } catch {
      setFormError("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading && activeToken) {
    return (
      <Container className="d-flex justify-content-center align-items-center" style={{ minHeight: "50vh" }}>
        <Spinner animation="border" variant="primary" />
      </Container>
    );
  }

  return (
    <Container className="profile-page">
      <section className="profile-panel" aria-labelledby="profile-title">
        <h1 id="profile-title">My Profile</h1>

        {formError && <Alert variant="danger">{formError}</Alert>}
        {formSuccess && <Alert variant="success">{formSuccess}</Alert>}

        <Form className="profile-content" onSubmit={handleSubmit}>
          <div className="profile-summary">
            <div>
              <h2>Account Settings</h2>
              <p className="profile-subtitle">
                View and update your account information.
              </p>
            </div>
            <div className="profile-mode-actions">
              {isEditing ? (
                <>
                  <Button type="submit" className="profile-save-button" disabled={isSubmitting}>
                    {isSubmitting ? "Saving..." : "Save"}
                  </Button>
                  <button
                    type="button"
                    className="profile-cancel-button"
                    onClick={() => {
                      setUsername(savedProfile.username);
                      setEmail(savedProfile.email);
                      setContactNumber(savedProfile.contactNumber);
                      setFieldErrors({});
                      setFormError("");
                      setFormSuccess("");
                      setIsEditing(false);
                    }}
                  >
                    Cancel
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  className="profile-edit-button"
                  onClick={() => {
                    setIsEditing(true);
                    setFieldErrors({});
                    setFormError("");
                    setFormSuccess("");
                  }}
                >
                  <span className="profile-edit-icon" aria-hidden="true" />
                  <span>Edit</span>
                </button>
              )}
            </div>
          </div>

          <div className="profile-details">
            <div className="profile-detail-row">
              <div className="profile-detail-label">
                <img src={userIcon} alt="" className="profile-detail-icon profile-detail-icon-user" />
                <span className="visually-hidden">Username</span>
              </div>
              <div className="profile-detail-value">
                {isEditing ? (
                  <>
                    <Form.Control
                      className="profile-inline-input"
                      aria-label="Username"
                      type="text"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      maxLength={FIELD_LIMITS.username}
                      isInvalid={!!fieldErrors.username}
                    />
                    <Form.Control.Feedback type="invalid">
                      {fieldErrors.username}
                    </Form.Control.Feedback>
                  </>
                ) : (
                  <span>{username || "-"}</span>
                )}
              </div>
            </div>

            <div className="profile-detail-row">
              <div className="profile-detail-label">
                <img src={emailIcon} alt="" className="profile-detail-icon" />
                <span className="visually-hidden">Email</span>
              </div>
              <div className="profile-detail-value">
                {isEditing ? (
                  <>
                    <Form.Control
                      className="profile-inline-input"
                      aria-label="Email"
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      maxLength={FIELD_LIMITS.email}
                      isInvalid={!!fieldErrors.email}
                    />
                    <Form.Control.Feedback type="invalid">
                      {fieldErrors.email}
                    </Form.Control.Feedback>
                  </>
                ) : (
                  <span>{email || "-"}</span>
                )}
              </div>
            </div>

            <div className="profile-detail-row">
              <div className="profile-detail-label">
                <img src={phoneIcon} alt="" className="profile-detail-icon profile-detail-icon-phone" />
                <span className="visually-hidden">Contact number</span>
              </div>
              <div className="profile-detail-value">
                {isEditing ? (
                  <>
                    <Form.Control
                      className="profile-inline-input"
                      aria-label="Contact number"
                      type="text"
                      value={contactNumber}
                      onChange={(e) => setContactNumber(e.target.value)}
                      maxLength={FIELD_LIMITS.contactNumber}
                      isInvalid={!!fieldErrors.contactNumber}
                    />
                    <Form.Control.Feedback type="invalid">
                      {fieldErrors.contactNumber}
                    </Form.Control.Feedback>
                  </>
                ) : (
                  <span>{contactNumber || "-"}</span>
                )}
              </div>
            </div>
          </div>
        </Form>

        <div className="profile-actions">
          <Button type="button" className="profile-action-button">
            Change Password
          </Button>
          {businessProfile ? (
            <Button
              type="button"
              className="profile-action-button profile-action-button-wide"
              onClick={() => navigate("/edit-business")}
            >
              Edit Business Profile
            </Button>
          ) : (
            <Button
              type="button"
              className="profile-action-button profile-action-button-wide"
              onClick={() => navigate("/register-business")}
            >
              Register Business Profile
            </Button>
          )}
        </div>
      </section>
    </Container>
  );
}
