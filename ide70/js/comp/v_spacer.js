function addResizerToTable(table) {
	var row = table.getElementsByTagName('tr')[0],
		cols = row ? row.children : undefined;
	if (!cols) return;

	table.style.overflow = 'hidden';

	for (var i = 0; i < cols.length - 1; i++) {
		var div = document.createElement('div');
		div.className = 'vspacer'
		cols[i].appendChild(div);
		cols[i].style.position = 'relative';
		setListeners(div);
	}

	function setListeners(div) {
		var pageX, curCol, nxtCol, curColWidth, nxtColWidth;

		div.addEventListener('mousedown', function(e) {
			curCol = e.target.parentElement;
			nxtCol = curCol.nextElementSibling;
			pageX = e.pageX;

			var padding = paddingDiff(curCol);

			curColWidth = curCol.offsetWidth - padding;
			if (nxtCol)
				nxtColWidth = nxtCol.offsetWidth - padding;
		});

		document.addEventListener('mousemove', function(e) {
			if (curCol) {
				var diffX = e.pageX - pageX;

				if (nxtCol)
					nxtCol.style.width = (nxtColWidth - (diffX)) + 'px';

				curCol.style.width = (curColWidth + diffX) + 'px';
			}
		});

		document.addEventListener('mouseup', function(e) {
			curCol = undefined;
			//nxtCol = undefined;
			//pageX = undefined;
			//nxtColWidth = undefined;
			//curColWidth = undefined
		});
	}

	function paddingDiff(col) {

		if (getStyleVal(col, 'box-sizing') == 'border-box') {
			return 0;
		}

		var padLeft = getStyleVal(col, 'padding-left');
		var padRight = getStyleVal(col, 'padding-right');
		return (parseInt(padLeft) + parseInt(padRight));

	}

	function getStyleVal(elm, css) {
		return (window.getComputedStyle(elm, null).getPropertyValue(css))
	}
};
