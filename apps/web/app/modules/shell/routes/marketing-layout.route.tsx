import { Outlet } from "react-router-dom";
import { Footer } from "../ui/Footer";
import { Header } from "../ui/Header";

export default function MarketingLayout() {
	return (
		<div className="min-h-screen bg-zinc-950 text-white">
			<Header />
			<div className="border-b border-white/10" />
			<div className="mx-auto max-w-6xl px-6 sm:px-8 py-16">
				<div className="mx-auto max-w-3xl">
					<Outlet />
				</div>
			</div>
			<Footer />
		</div>
	);
}