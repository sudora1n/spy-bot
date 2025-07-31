import { Link as RouterLink } from "react-router-dom";
import type { FC, ReactNode } from "react";

interface LinkProps {
  to: string;
  children: ReactNode;
  className?: string;
}

export const Link: FC<LinkProps> = ({ to, children, className = "" }) => {
  return (
    <RouterLink
      to={to}
      className={`block ${className}`}
      style={{ textDecoration: "none", color: "inherit" }}
    >
      {children}
    </RouterLink>
  );
};
