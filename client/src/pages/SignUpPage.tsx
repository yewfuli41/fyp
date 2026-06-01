import { useState } from "react";
import type { FormEvent } from "react";
import { Alert, Button, Container, Form } from "react-bootstrap";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import PasswordInput from "../components/PasswordInput";
import { signUp } from "../services/SignUpService";
import "../styles/auth.css";

type FieldErrors = Record<string, string>;

export default function SignUpPage() {
  const navigate = useNavigate();
  const { login } = useAuth();

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [contactNumber, setContactNumber] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFieldErrors({});
    setFormError("");

    const nextErrors: FieldErrors = {};
    if (password !== confirmPassword) {
      nextErrors.confirmPassword = "Passwords do not match";
    }

    setIsSubmitting(true);

    try {
      const result = await signUp(username, email, contactNumber, password);

      if (result.errors?.length) {
        const validationErrors = result.errors[0]?.extensions?.validationErrors;
        if (validationErrors?.length) {
          for (const item of validationErrors) {
            nextErrors[item.field] = item.message;
          }
        } else {
          setFormError(result.errors[0]?.message ?? "Sign up failed");
        }
      } else {
        const payload = result.data?.signUp;
        if (!payload) {
          setFormError("Sign up failed");
        } else if (Object.keys(nextErrors).length === 0) {
          login(payload.token, {
            userId: Number(payload.user.userId),
            username: payload.user.username,
            email: payload.user.email,
            contactNumber: payload.user.contactNumber,
          });
          navigate("/");
          return;
        }
      }

      if (Object.keys(nextErrors).length > 0) {
        setFieldErrors(nextErrors);
      }
    } catch {
      if (Object.keys(nextErrors).length > 0) {
        setFieldErrors(nextErrors);
      }
      setFormError("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Container className="auth-page py-5">
      <div className="auth-card">
        <h1 className="mb-4">Sign up</h1>

        {formError && <Alert variant="danger">{formError}</Alert>}

        <Form onSubmit={handleSubmit}>
          <Form.Group className="mb-3" controlId="username">
            <Form.Label>Username</Form.Label>
            <Form.Control
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              isInvalid={!!fieldErrors.username}
            />
            <Form.Control.Feedback type="invalid">
              {fieldErrors.username}
            </Form.Control.Feedback>
          </Form.Group>

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

          <Form.Group className="mb-3" controlId="contactNumber">
            <Form.Label>Contact number</Form.Label>
            <Form.Control
              type="text"
              value={contactNumber}
              onChange={(e) => setContactNumber(e.target.value)}
              isInvalid={!!fieldErrors.contactNumber}
            />
            <Form.Control.Feedback type="invalid">
              {fieldErrors.contactNumber}
            </Form.Control.Feedback>
          </Form.Group>

          <PasswordInput
            controlId="password"
            label="Password"
            value={password}
            onChange={setPassword}
            isInvalid={!!fieldErrors.password}
            errorMessage={fieldErrors.password}
            className="mb-3"
          />

          <PasswordInput
            controlId="confirmPassword"
            label="Confirm password"
            value={confirmPassword}
            onChange={setConfirmPassword}
            isInvalid={!!fieldErrors.confirmPassword}
            errorMessage={fieldErrors.confirmPassword}
            className="mb-4"
          />

          <Button type="submit" variant="primary" className="w-100" disabled={isSubmitting}>
            {isSubmitting ? "Signing up..." : "Sign up"}
          </Button>
        </Form>

        <p className="auth-footer mt-3 mb-0">
          Already have an account? <Link to="/login">Log in</Link>
        </p>
      </div>
    </Container>
  );
}
