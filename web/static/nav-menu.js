(() => {
	const init = () => {
		const nav = document.querySelector("[data-mobile-nav]");
		if (!nav) {
			return;
		}

		const toggle = nav.querySelector("[data-nav-toggle]");
		const menu = nav.querySelector("[data-nav-menu]");
		if (!toggle || !menu) {
			return;
		}

		const close = () => {
			nav.classList.remove("open");
			toggle.setAttribute("aria-expanded", "false");
		};

		const open = () => {
			nav.classList.add("open");
			toggle.setAttribute("aria-expanded", "true");
		};

		toggle.addEventListener("click", () => {
			if (nav.classList.contains("open")) {
				close();
				return;
			}
			open();
		});

		document.addEventListener("click", (event) => {
			if (!nav.classList.contains("open")) {
				return;
			}
			if (nav.contains(event.target)) {
				return;
			}
			close();
		});

		document.addEventListener("keydown", (event) => {
			if (event.key === "Escape") {
				close();
			}
		});

		menu.querySelectorAll("a").forEach((link) => {
			link.addEventListener("click", close);
		});

		window.addEventListener("resize", () => {
			if (window.innerWidth > 800) {
				close();
			}
		});
	};

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", init, { once: true });
		return;
	}

	init();
})();
