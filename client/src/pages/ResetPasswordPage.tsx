import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { Alert, Button, Container, Form, Modal } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import PasswordInput from "../components/PasswordInput";
import { resetPassword } from "../services/ResetPasswordService";
import { shouldClearPasswordError } from "../utils/fieldLimits";
import { parseGraphQLErrors, type FieldErrors } from "../utils/graphqlErrors";
import "../styles/auth.css";

export default function ResetPasswordPage() {
  const navigate = useNavigate();
  const { token, user, login } = useAuth();
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (user && !user.mustResetPassword) {
      navigate("/", { replace: true });
    }
  }, [navigate, user]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFieldErrors({});
    setFormError("");

    const nextErrors: FieldErrors = {};
    if (newPassword.length < 8) {
      nextErrors.password = "Password must be at least 8 characters";
    }
    if (newPassword !== confirmPassword) {
      nextErrors.confirmPassword = "Passwords do not match";
    }

    if (!token || !user) {
      setFormError("Session expired. Please log in again.");
      return;
    }

    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      return;
    }

    setIsSubmitting(true);

    try {
      const result = await resetPassword(token, newPassword);
      const parsed = parseGraphQLErrors(result, "Password reset failed");
      if (parsed.hasErrors) {
        setFieldErrors(parsed.fieldErrors);
        setFormError(parsed.formError);
        return;
      }

      login(token, { ...user, mustResetPassword: false });
      navigate("/", { replace: true });
    } catch {
      setFormError("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Container className="auth-page py-5">
      <Modal show centered backdrop="static" keyboard={false}>
        <Modal.Header>
          <Modal.Title>Reset password</Modal.Title>
        </Modal.Header>
        <Form onSubmit={handleSubmit}>
          <Modal.Body>
            <Alert variant="info">
              Please replace your temporary staff password before continuing.
            </Alert>
            {formError && <Alert variant="danger">{formError}</Alert>}
            <PasswordInput
              controlId="newPassword"
              label="New password"
              value={newPassword}
              onChange={(value) => {
                setNewPassword(value);
                if (shouldClearPasswordError(fieldErrors.password, value)) {
                  setFieldErrors((prev) => ({ ...prev, password: "" }));
                }
              }}
              isInvalid={!!fieldErrors.password}
              errorMessage={fieldErrors.password}
              className="mb-3"
            />
            <PasswordInput
              controlId="confirmPassword"
              label="Confirm password"
              value={confirmPassword}
              onChange={(value) => {
                setConfirmPassword(value);
                setFieldErrors((prev) => {
                  const next = { ...prev };
                  delete next.confirmPassword;
                  return next;
                });
              }}
              isInvalid={!!fieldErrors.confirmPassword}
              errorMessage={fieldErrors.confirmPassword}
            />
          </Modal.Body>
          <Modal.Footer>
            <Button type="submit" variant="primary" disabled={isSubmitting} className="w-100">
              {isSubmitting ? "Resetting..." : "Reset password"}
            </Button>
          </Modal.Footer>
        </Form>
      </Modal>
    </Container>
  );
}
