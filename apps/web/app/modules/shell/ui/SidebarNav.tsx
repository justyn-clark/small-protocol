import { NavLink } from "react-router-dom";
import type { DocSection } from "~/modules/docs/lib/nav";

export function SidebarNav({
	nav,
	activeSlug,
}: {
	nav: DocSection[];
	activeSlug: string;
}) {
	return (
		<nav className="space-y-8">
			{nav.map((section) => (
				<div key={section.id}>
					<div className="mb-2 text-xs font-semibold uppercase tracking-wide text-white/50">
						{section.title}
					</div>
					<ul className="space-y-1">
						{section.items.map((item) => {
							const to = item.slug === "docs" ? "/docs" : `/docs/${item.slug}`;
							return (
								<li key={item.slug}>
									<NavLink
										to={to}
										className={({ isActive }) =>
											[
												"block rounded-lg px-3 py-2 text-sm",
												isActive || activeSlug === item.slug
													? "bg-white/10 text-white"
													: "text-white/70 hover:bg-white/5 hover:text-white",
											].join(" ")
										}
									>
										{item.title}
									</NavLink>
								</li>
							);
						})}
					</ul>
				</div>
			))}
		</nav>
	);
}
