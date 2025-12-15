import type * as T from "react-router/route-module";

type Module = typeof import("../catch-all.route.js");
type RootModule = typeof import("../../root.js");

export type Info = {
	parents: [Parent0];
	id: "routes/catch-all";
	file: "./routes/catch-all.route.tsx";
	path: "/*";
	params: { "*": string } & { [key: string]: string | undefined };
	module: Module;
	loaderData: T.CreateLoaderData<Module>;
	actionData: T.CreateActionData<Module>;
};

type Parent0 = {
	parents: [];
	id: "root";
	file: "root.tsx";
	path: "";
	params: {} & { [key: string]: string | undefined };
	module: RootModule;
	loaderData: T.CreateLoaderData<RootModule>;
	actionData: T.CreateActionData<RootModule>;
};

export namespace Route {
	export type LinkDescriptors = T.LinkDescriptors;
	export type LinksFunction = () => LinkDescriptors;

	export type MetaArgs = T.CreateMetaArgs<Info>;
	export type MetaDescriptors = T.MetaDescriptors;
	export type MetaFunction = (args: MetaArgs) => MetaDescriptors;

	export type HeadersArgs = T.HeadersArgs;
	export type HeadersFunction = (args: HeadersArgs) => Headers | HeadersInit;

	export type unstable_MiddlewareFunction =
		T.CreateServerMiddlewareFunction<Info>;
	export type unstable_ClientMiddlewareFunction =
		T.CreateClientMiddlewareFunction<Info>;
	export type LoaderArgs = T.CreateServerLoaderArgs<Info>;
	export type ClientLoaderArgs = T.CreateClientLoaderArgs<Info>;
	export type ActionArgs = T.CreateServerActionArgs<Info>;
	export type ClientActionArgs = T.CreateClientActionArgs<Info>;

	export type HydrateFallbackProps = T.CreateHydrateFallbackProps<Info>;
	export type ComponentProps = T.CreateComponentProps<Info>;
	export type ErrorBoundaryProps = T.CreateErrorBoundaryProps<Info>;
}
