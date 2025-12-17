import protocol from "~/modules/protocol/small.protocol.json";

export async function loader() {
	return Response.json(protocol, {
		headers: {
			"Content-Type": "application/json",
		},
	});
}
