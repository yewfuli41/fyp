import { useState, useEffect } from "react";
import type { FormEvent } from "react";
import { Alert, Button, Container, Form } from "react-bootstrap";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import PasswordInput from "../components/PasswordInput";
import { logIn } from "../services/LogInService";
import { parseGraphQLErrors } from "../utils/graphqlErrors";
import { shouldClearEmailError, shouldClearPasswordError } from "../utils/fieldLimits";
import "../styles/auth.css";

export default function LogInPage() {
  const navigate = useNavigate();
  const { login, isLoggedIn, user } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isLoggedIn) {
      navigate(user?.staffProfile && user.mustResetPassword ? "/reset-password" : "/");
    }
    const msg = sessionStorage.getItem("authMessage");
    if(msg){
      setFormError(msg);
      sessionStorage.removeItem("authMessage");
    }
  }, [isLoggedIn, user, navigate]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFieldErrors({});
    setFormError("");
    
    setIsSubmitting(true);

    try {
      const result = await logIn(email, password);

      const parsed = parseGraphQLErrors(result, "Log in failed");
      if (parsed.hasErrors) {
        if (parsed.extensions?.lockedUntil) {
          const lockedUntil = new Date(String(parsed.extensions.lockedUntil));
          setFormError(`Account is locked until ${lockedUntil.toLocaleString()}`);
        } else {
          setFieldErrors(parsed.fieldErrors);
          setFormError(parsed.formError);
        }
        return;
      }

      const payload = result.data?.logIn;
      if (!payload) {
        setFormError("Log in failed");
        return;
      }

      const loggedInUser = {
        userId: Number(payload.user.userId),
        username: payload.user.username,
        email: payload.user.email,
        contactNumber: payload.user.contactNumber,
        mustResetPassword: payload.user.mustResetPassword,
        businessProfile: payload.user.businessProfile,
        staffProfile: payload.user.staffProfile,
      };

      login(payload.token, loggedInUser);
      //navigate(loggedInUser.staffProfile && loggedInUser.mustResetPassword ? "/reset-password" : "/");
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
              onChange={(e) => {
                setEmail(e.target.value)
                if (shouldClearEmailError(fieldErrors.email, e.target.value))
                  setFieldErrors(prev => ({ ...prev, email: "" }))
              }}
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
            onChange={(value) => {
              setPassword(value)
              if (shouldClearPasswordError(fieldErrors.password, value))
                setFieldErrors(prev => ({ ...prev, password: "" }))
            }}
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
