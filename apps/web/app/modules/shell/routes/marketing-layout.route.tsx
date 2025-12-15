import { Outlet } from "react-router-dom";
import { Footer } from "../ui/Footer";
import { Header } from "../ui/Header";

export default function MarketingLayout() {
	return (
		<div className="min-h-screen bg-zinc-950 text-white">
			<Header />
			<div className="mx-auto max-w-6xl px-4 py-10">
				<Outlet />
			</div>
			<Footer />
		</div>
	);
}
