import type { LoaderFunctionArgs } from "react-router-dom";
import { Outlet, useLoaderData } from "react-router-dom";
import { getDocMeta } from "~/modules/docs/lib/meta";
import { DOCS_NAV } from "~/modules/docs/lib/nav";
import { Footer } from "../ui/Footer";
import { Header } from "../ui/Header";
import { SidebarNav } from "../ui/SidebarNav";

function slugFromRequest(request: Request) {
	const url = new URL(request.url);
	const parts = url.pathname.split("/").filter(Boolean);
	// /docs -> docs
	// /docs/:slug -> slug
	return parts.length === 1 ? "docs" : parts[1];
}

export async function loader({ request }: LoaderFunctionArgs) {
	const slug = slugFromRequest(request);
	const meta = getDocMeta(slug);
	return {
		nav: DOCS_NAV,
		activeSlug: slug,
		title: meta.title,
		description: meta.description,
	};
}

export default function DocsLayout() {
	const data = useLoaderData() as {
		nav: typeof DOCS_NAV;
		activeSlug: string;
		title: string;
		description?: string;
	};

	return (
		<div className="min-h-screen bg-zinc-950 text-white">
			<Header />
			<div className="border-b border-white/10" />
			<div className="mx-auto flex max-w-7xl gap-10 px-4 py-10">
				<aside className="hidden w-72 shrink-0 lg:block">
					<SidebarNav nav={data.nav} activeSlug={data.activeSlug} />
				</aside>
				<main className="min-w-0 flex-1">
					<div className="max-w-3xl">
						<div className="mb-8">
							{data.description ? (
								<p className="text-white/70">{data.description}</p>
							) : null}
						</div>
						<Outlet />
					</div>
				</main>
			</div>
			<Footer />
		</div>
	);
}
