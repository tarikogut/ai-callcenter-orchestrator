import React from 'react';
import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive';
}

export const Badge: React.FC<BadgeProps> = ({
  className,
  variant = 'default',
  children,
  ...props
}) => {
  const variants = {
    default: 'bg-primary/10 text-primary border-primary/20',
    secondary: 'bg-secondary text-secondary-foreground border-border',
    outline: 'border border-border text-foreground',
    success: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20',
    warning: 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20',
    destructive: 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/20',
  };

  return (
    <span
      className={twMerge(
        clsx(
          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold border transition-colors',
          variants[variant],
          className
        )
      )}
      {...props}
    >
      {children}
    </span>
  );
};
