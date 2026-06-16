import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { api } from "@/services/api";
import { useAuthStore } from "@/store/auth";
import { Button, Input, Label } from "@/components/ui/primitives";
import { AuthShell } from "./AuthShell";

const schema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
});
type Form = z.infer<typeof schema>;

export function LoginPage() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);
  const [error, setError] = useState("");
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { email: "demo@metrics.dev", password: "demo123456" },
  });

  const onSubmit = async (data: Form) => {
    setError("");
    try {
      const res = await api.login(data.email, data.password);
      setAuth(res.access_token, res.refresh_token, res.user);
      navigate("/");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Login failed");
    }
  };

  return (
    <AuthShell
      title="Welcome back"
      subtitle="Sign in to your observability workspace"
      footer={
        <>
          No account?{" "}
          <Link to="/register" className="text-primary hover:underline">
            Create one
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" {...register("email")} />
          {errors.email && (
            <p className="text-xs text-danger">{errors.email.message}</p>
          )}
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="password">Password</Label>
          <Input id="password" type="password" {...register("password")} />
          {errors.password && (
            <p className="text-xs text-danger">{errors.password.message}</p>
          )}
        </div>
        {error && <p className="text-xs text-danger">{error}</p>}
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "Signing in…" : "Sign in"}
        </Button>
        <p className="text-center text-xs text-fg-muted">
          Demo: demo@metrics.dev / demo123456
        </p>
      </form>
    </AuthShell>
  );
}
