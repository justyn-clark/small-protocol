import { NavLink } from "react-router-dom";

export function Header() {
	return (
		<header className="sticky top-0 z-40 border-b border-white/10 bg-zinc-950/80 backdrop-blur">
			<div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
				<NavLink to="/" className="text-sm font-semibold tracking-tight">
					Agent-Legible CMS
				</NavLink>
				<nav className="flex items-center gap-6 text-sm">
					<NavLink
						to="/docs"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Docs
					</NavLink>
					<NavLink
						to="/pricing"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Pricing
					</NavLink>
					<NavLink
						to="/about"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						About
					</NavLink>
					<NavLink
						to="/contact"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Contact
					</NavLink>
				</nav>
			</div>
		</header>
	);
}
