import { useState } from "react";
import type { FormEvent } from "react";
import { Alert, Button, Container, Form } from "react-bootstrap";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import PasswordInput from "../components/PasswordInput";
import { logIn } from "../services/LogInService";
import "../styles/auth.css";

type FieldErrors = Record<string, string>;

export default function LogInPage() {
  const navigate = useNavigate();
  const { login } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFieldErrors({});
    setFormError("");

    setIsSubmitting(true);

    try {
      const result = await logIn(email, password);

      if (result.errors?.length) {
        const firstError = result.errors[0];
        const validationErrors = firstError?.extensions?.validationErrors;

        if (validationErrors?.length) {
          const nextErrors: FieldErrors = {};
          for (const item of validationErrors) {
            nextErrors[item.field] = item.message;
          }
          setFieldErrors(nextErrors);
        } else if (firstError?.extensions?.lockedUntil) {
          const lockedUntil = new Date(String(firstError.extensions.lockedUntil));
          setFormError(
            `Account is locked until ${lockedUntil.toLocaleString()}`,
          );
        } else {
          setFormError(firstError?.message ?? "Log in failed");
        }
        return;
      }

      const payload = result.data?.logIn;
      if (!payload) {
        setFormError("Log in failed");
        return;
      }

      login(payload.token, {
        userId: Number(payload.user.userId),
        username: payload.user.username,
        email: payload.user.email,
        contactNumber: payload.user.contactNumber,
      });
      navigate("/");
    } catch {
      setFormError("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Container className="auth-page py-5">
      <div className="auth-card">
        <h1 className="mb-4">Log in</h1>

        {formError && <Alert variant="danger">{formError}</Alert>}

        <Form onSubmit={handleSubmit}>
          <Form.Group className="mb-3" controlId="email">
            <Form.Label>Email</Form.Label>
            <Form.Control
              type="text"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              isInvalid={!!fieldErrors.email}
            />
            <Form.Control.Feedback type="invalid">
              {fieldErrors.email}
            </Form.Control.Feedback>
          </Form.Group>

          <PasswordInput
            controlId="password"
            label="Password"
            value={password}
            onChange={setPassword}
            isInvalid={!!fieldErrors.password}
            errorMessage={fieldErrors.password}
            className="mb-4"
          />

          <Button type="submit" variant="primary" className="w-100" disabled={isSubmitting}>
            {isSubmitting ? "Logging in..." : "Log in"}
          </Button>
        </Form>

        <p className="auth-footer mt-3 mb-0">
          Don&apos;t have an account? <Link to="/signup">Sign up</Link>
        </p>
      </div>
    </Container>
  );
}
