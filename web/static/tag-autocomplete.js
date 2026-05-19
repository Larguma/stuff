(() => {
	const isCoarsePointer = () => {
		return window.matchMedia && window.matchMedia("(pointer: coarse)").matches;
	};

	const normalizePrefix = (parts) => {
		const cleaned = parts.map((part) => part.trim()).filter((part) => part.length > 0);
		return cleaned.join(", ");
	};

	const parseInput = (mode, value) => {
		if (mode === "tags") {
			const parts = value.split(",");
			const last = parts.pop() || "";
			const prefix = normalizePrefix(parts);
			return { prefix, last };
		}
		return { prefix: "", last: value };
	};

	const formatSelection = (mode, prefix, value) => {
		if (mode === "tags") {
			return prefix ? `${prefix}, ${value}` : value;
		}
		return value;
	};

	const updateDatalist = (mode, list, allOptions, prefixValue, filterValue) => {
		const term = filterValue.trim().toLowerCase();
		const prefix = mode === "tags" && prefixValue ? `${prefixValue}, ` : "";
		list.innerHTML = "";
		for (const optionValue of allOptions) {
			if (term && !optionValue.toLowerCase().startsWith(term)) {
				continue;
			}
			const option = document.createElement("option");
			option.value = prefix + optionValue;
			list.appendChild(option);
		}
	};

	const setupCustomList = (input, allOptions, mode) => {
		const label = input.closest("label");
		if (label) {
			label.classList.add("autocomplete-field");
		}

		const container = document.createElement("div");
		container.className = "tag-suggestions";
		container.hidden = true;
		if (label) {
			label.appendChild(container);
		} else {
			input.insertAdjacentElement("afterend", container);
		}

		const render = (prefixValue, filterValue) => {
			const term = filterValue.trim().toLowerCase();
			const matches = allOptions.filter((optionValue) => {
				if (!term) {
					return true;
				}
				return optionValue.toLowerCase().startsWith(term);
			}).slice(0, 8);

			container.innerHTML = "";
			if (matches.length === 0) {
				container.hidden = true;
				return;
			}

			for (const optionValue of matches) {
				const button = document.createElement("button");
				button.type = "button";
				button.className = "tag-suggestion";
				button.textContent = optionValue;
				button.dataset.value = optionValue;
				container.appendChild(button);
			}
			container.hidden = false;
			input.dataset.autocompletePrefix = prefixValue || "";
		};

		const handleInput = () => {
			const { prefix, last } = parseInput(mode, input.value);
			render(prefix, last);
		};

		input.addEventListener("input", handleInput);
		input.addEventListener("focus", handleInput);
		input.addEventListener("blur", () => {
			setTimeout(() => {
				container.hidden = true;
			}, 150);
		});
		container.addEventListener("click", (event) => {
			const target = event.target.closest("button.tag-suggestion");
			if (!target) {
				return;
			}
			const prefix = input.dataset.autocompletePrefix || "";
			const value = target.dataset.value || "";
			input.value = formatSelection(mode, prefix, value);
			container.hidden = true;
			input.focus();
		});
	};

	const initInput = (input, mode) => {
		const listId = input.getAttribute("list");
		if (!listId) {
			return;
		}

		const list = document.getElementById(listId);
		if (!list) {
			return;
		}

		const allOptions = Array.from(list.options)
			.map((option) => option.value)
			.filter((value) => value.length > 0);

		if (isCoarsePointer()) {
			input.removeAttribute("list");
			setupCustomList(input, allOptions, mode);
			return;
		}

		const refreshList = () => {
			const { prefix, last } = parseInput(mode, input.value);
			updateDatalist(mode, list, allOptions, prefix, last);
		};

		input.addEventListener("input", refreshList);
		refreshList();
	};

	document.addEventListener("DOMContentLoaded", () => {
		const fields = [
			{ selector: "input.tag-input[list]", mode: "tags" },
			{ selector: "input.location-input[list]", mode: "single" },
		];
		fields.forEach((field) => {
			document.querySelectorAll(field.selector).forEach((input) => {
				initInput(input, field.mode);
			});
		});
	});
})();
