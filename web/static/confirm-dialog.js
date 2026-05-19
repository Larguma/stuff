(() => {
	const ready = () => {
		const modal = document.querySelector("[data-confirm-modal]");
		if (!modal) {
			return;
		}

		const messageNode = modal.querySelector("[data-confirm-message]");
		const acceptButton = modal.querySelector("[data-confirm-accept]");
		const cancelButton = modal.querySelector("[data-confirm-cancel]");
		const backdrop = modal.querySelector("[data-confirm-close]");
		let activeForm = null;
		let activeSubmitter = null;

		const close = () => {
			modal.hidden = true;
			modal.setAttribute("aria-hidden", "true");
			activeForm = null;
			activeSubmitter = null;
		};

		const open = (form, submitter) => {
			activeForm = form;
			activeSubmitter = submitter || null;
			if (messageNode) {
				messageNode.textContent = form.dataset.confirm || "";
			}
			if (acceptButton) {
				acceptButton.textContent = (submitter && submitter.textContent && submitter.textContent.trim()) || acceptButton.textContent;
			}
			modal.hidden = false;
			modal.setAttribute("aria-hidden", "false");
			if (acceptButton) {
				acceptButton.focus();
			}
		};

		document.addEventListener("submit", (event) => {
			const form = event.target;
			if (!(form instanceof HTMLFormElement)) {
				return;
			}
			if (!form.dataset.confirm) {
				return;
			}
			if (form.dataset.confirmed === "1") {
				delete form.dataset.confirmed;
				return;
			}
			event.preventDefault();
			open(form, event.submitter || null);
		}, true);

		const confirmAction = () => {
			if (!activeForm) {
				close();
				return;
			}
			const form = activeForm;
			const submitter = activeSubmitter;
			form.dataset.confirmed = "1";
			close();
			if (submitter && typeof submitter.click === "function") {
				submitter.click();
				return;
			}
			if (typeof form.requestSubmit === "function") {
				form.requestSubmit();
				return;
			}
			form.submit();
		};

		if (acceptButton) {
			acceptButton.addEventListener("click", confirmAction);
		}
		if (cancelButton) {
			cancelButton.addEventListener("click", close);
		}
		if (backdrop) {
			backdrop.addEventListener("click", close);
		}
		document.addEventListener("keydown", (event) => {
			if (!modal.hidden && event.key === "Escape") {
				close();
			}
		});
	};

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", ready, { once: true });
		return;
	}

	ready();
})();