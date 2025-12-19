import { useEffect, useState } from "react";
import { NavLink } from "react-router-dom";

const navLinks = [
	{ to: "/", label: "Overview", mobileLabel: "Overview" },
	{ to: "/blog/ai-needs-execution", label: "Essay", mobileLabel: "Essay" },
	{ to: "/spec", label: "Spec", mobileLabel: "Spec" },
	{ to: "/compliance", label: "Compliance", mobileLabel: "Compliance" },
	{
		to: "/reference-workflow",
		label: "Reference Workflow",
		mobileLabel: "Workflow",
	},
	{ to: "/about", label: "About", mobileLabel: "About" },
];

export function Header() {
	const [isMenuOpen, setIsMenuOpen] = useState(false);

	const closeMenu = () => setIsMenuOpen(false);

	useEffect(() => {
		const handleEscape = (e: KeyboardEvent) => {
			if (e.key === "Escape" && isMenuOpen) {
				setIsMenuOpen(false);
			}
		};

		if (isMenuOpen) {
			document.addEventListener("keydown", handleEscape);
			return () => document.removeEventListener("keydown", handleEscape);
		}
	}, [isMenuOpen]);

	return (
		<header className="sticky top-0 z-40 border-b border-white/10 bg-zinc-950/80 backdrop-blur">
			<div className="relative mx-auto flex max-w-6xl items-center justify-between px-6 py-3 sm:px-8">
				<NavLink to="/" className="text-sm font-semibold tracking-tight">
					SMALL
				</NavLink>

				{/* Desktop Navigation */}
				<nav className="hidden items-center gap-6 text-sm md:flex">
					{navLinks.map((link) => (
						<NavLink
							key={link.to}
							to={link.to}
							className={({ isActive }) =>
								`whitespace-nowrap ${isActive ? "text-white" : "text-white/70 hover:text-white"
								}`
							}
						>
							{link.label}
						</NavLink>
					))}
				</nav>

				{/* Mobile Menu Button */}
				<button
					type="button"
					className="md:hidden p-2 text-white/70 hover:text-white"
					onClick={() => setIsMenuOpen(!isMenuOpen)}
					aria-label="Toggle menu"
					aria-expanded={isMenuOpen}
				>
					<svg
						className="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						strokeWidth={2}
					>
						{isMenuOpen ? (
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								d="M6 18L18 6M6 6l12 12"
							/>
						) : (
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								d="M4 6h16M4 12h16M4 18h16"
							/>
						)}
					</svg>
				</button>

				{/* Mobile Menu Panel */}
				{isMenuOpen && (
					<div className="absolute left-0 top-full w-full border-b border-white/10 bg-zinc-950/95 backdrop-blur">
						<nav className="flex flex-col">
							{navLinks.map((link, index) => (
								<NavLink
									key={link.to}
									to={link.to}
									onClick={closeMenu}
									className={({ isActive }) =>
										`px-6 py-3 text-base sm:px-8 ${index < navLinks.length - 1
											? "border-b border-white/5"
											: ""
										} ${isActive
											? "text-white"
											: "text-white/70 hover:text-white"
										}`
									}
								>
									{link.mobileLabel}
								</NavLink>
							))}
						</nav>
					</div>
				)}
			</div>
		</header>
	);
}
