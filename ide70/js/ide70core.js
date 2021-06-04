var _pEventType='et'; 
var _pCompId='cid';
var _pCompValue='cval';
var _pFocCompId='fcid';
var _pMouseWX='mwx';
var _pMouseWY='mwy'; 
var _pMouseX='mx';
var _pMouseY='my';
var _pMouseBtn='mb';
var _pModKeys='mk';
var _pKeyCode='kc'; 
var _pScrollTop='sctp'; 

// Modifier key masks
var _modKeyAlt=1; 
var _modKeyCtlr=2; 
var _modKeyMeta=4; 
var _modKeyShift=8; 

// Event response action consts
var _eraNoAction=0; 
var _eraReloadWin=1; 
var _eraDirtyComps=2; 
var _eraFocusComp=3; 
var _eraDirtyAttrs=4;
var _eraDirtyProps=5;
var _eraApplyToParent=6; 
var _eraScrollDownComp=7;
var _eraExecuteCompFunc=8;
var _eraForwardToParent=9;

function createXmlHttp() {
	if (window.XMLHttpRequest) // IE7+, Firefox, Chrome, Opera, Safari
		return new XMLHttpRequest();
	else // IE6, IE5
		return new ActiveXObject("Microsoft.XMLHTTP");
}

function wrapFormFileInput(fileInput){
	var fd = new FormData();
	fd.append('file', fileInput.files[0]); 
	return fd;
}

function seFile(xhr, etype, compId, fd) {
	if (etype != null)
		fd.append(_pEventType, etype);
	if (compId != null)
		fd.append(_pCompId, compId);
	if (document.activeElement.id != null)
		fd.append(_pFocCompId, document.activeElement.id);
		
	xhr.send(fd);
}

// Send event
// optional filter function filters inappropriate events
function se(event, etype, compId, compValue, async, filter) {
	if (filter && !filter(document.getElementById(compId))) {
		return;
	}
	
	if (etype == null) {
		etype = 'on' + event.type;
	}

	var xhr = createXmlHttp();
	
	xhr.onreadystatechange = function() {
		if (xhr.readyState == 4 && xhr.status == 200) {
			procEresp(xhr.responseText.split("|"));
		}
	}
	
	xhr.open("POST", _pathEvent + (async?"a":""), true); // asynch call
	
	if (compValue instanceof FormData) {
		return seFile(xhr, etype, compId, compValue)
	}
	
	xhr.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
	
	var data="";
	
	if (etype != null)
		data += "&" + _pEventType + "=" + etype;
	// remove special event types
	etype = etype.split('_')[0];
	if (compId != null)
		data += "&" + _pCompId + "=" + compId;
	if (compValue != null)
		data += "&" + _pCompValue + "=" + compValue;
	if (document.activeElement.id != null)
		data += "&" + _pFocCompId + "=" + document.activeElement.id;
	
	if(etype == 'onscroll') {
		var comp = document.getElementById(compId);
		if (comp) {
			data += "&" + _pScrollTop + "=" + comp.scrollTop;
		}
	}
	
	if (event != null) {
		if (event.clientX != null) {
			// Mouse data
			var x = event.clientX, y = event.clientY;
			data += "&" + _pMouseWX + "=" + x;
			data += "&" + _pMouseWY + "=" + y;
			//var parent = document.getElementById(compId);
			var parent = event.target;
			do {
				x -= parent.offsetLeft;
				y -= parent.offsetTop;
			} while (parent = parent.offsetParent);
			data += "&" + _pMouseX + "=" + x;
			data += "&" + _pMouseY + "=" + y;
			data += "&" + _pMouseBtn + "=" + (event.button < 4 ? event.button : 1); // IE8 and below uses 4 for middle btn
		}
		
		var modKeys;
		modKeys += event.altKey ? _modKeyAlt : 0;
		modKeys += event.ctlrKey ? _modKeyCtlr : 0;
		modKeys += event.metaKey ? _modKeyMeta : 0;
		modKeys += event.shiftKey ? _modKeyShift : 0;
		data += "&" + _pModKeys + "=" + modKeys;
		data += "&" + _pKeyCode + "=" + (event.which ? event.which : event.keyCode);
	}
	console.log("send data: "+data)
	
	xhr.send(data);
}

function procEresp(actions) {
	
	if (actions.length == 0) {
		window.alert("No response received!");
		return;
	}
	for (var i = 0; i < actions.length; i++) {
		var n = actions[i].split(",");

		switch (parseInt(n[0])) {
		case _eraDirtyComps:
			for (var j = 1; j < n.length; j++)
				rerenderComp(n[j]);
			break;
		case _eraFocusComp:
			if (n.length > 1)
				focusComp(parseInt(n[1]))
			break;
		case _eraScrollDownComp:
			if (n.length > 1)
				scrollDownComp(parseInt(n[1]),parseInt(n[2]))
			break;
		case _eraNoAction:
			break;
		case _eraReloadWin:
			if (n.length > 1 && n[1].length > 0) {
			    if (n.length > 2 && n[2].length > 0) {
			        var e = document.getElementById(n[2]);
			        if(e) {
			            e.src = _pathApp + n[1];
			        }
			    } else {
				    window.location.href = _pathApp + n[1];
			    }
			} else {
				window.location.reload(true); // force reload
			}
			break;
		case _eraDirtyAttrs:
			for (var j = 1; j+2 < n.length; j+=3)
				replaceCompAttr(n[j], n[j+1], n[j+2]);
			break;
		case _eraDirtyProps:
			for (var j = 1; j+2 < n.length; j+=3)
				replaceCompProp(n[j], n[j+1], n[j+2]);
			break;
		case _eraExecuteCompFunc:
			if(n.length >= 3) {
				executeCompFunc(n[1], n[2], ...n.slice(3));
			}
			break;
		case _eraApplyToParent:
			window.parent.procEresp(actions.slice(i+1));
			return;
		case _eraForwardToParent:
		    window.parent.se(null, n[1], n[2])
		    return;
		default:
			window.alert("Unknown response code:" + n[0]);
			break;
		}
	}
}

function rerenderComp(compId) {
	var e = document.getElementById(compId);
	if (!e) // Component removed or not visible (e.g. on inactive tab of TabPanel)
		return;
	
	var xhr = createXmlHttp();
	
	xhr.onreadystatechange = function() {
		if (xhr.readyState == 4 && xhr.status == 200) {
			// Remember focused comp which might be replaced here:
			var focusedCompId = document.activeElement.id;
			e.outerHTML = xhr.responseText;
			focusComp(focusedCompId);
			
			// Inserted JS code is not executed automatically, do it manually:
			// Have to "re-get" element by compId!
			var scripts = document.getElementById(compId).getElementsByTagName("script");
			for (var i = 0; i < scripts.length; i++) {
				eval(scripts[i].innerText);
			}
		}
	}
	
	xhr.open("POST", _pathRenderComp, false); // synch call (if async, browser specific DOM rendering errors may arise)
	xhr.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
	
	xhr.send(_pCompId + "=" + compId);
}

function replaceCompAttr(compId, attrName, attrValue) {
	var e = document.getElementById(compId);
	if (!e) // Component removed or not visible (e.g. on inactive tab of TabPanel)
		return;
	
	e.setAttribute(attrName, attrValue)
}

function replaceCompProp(compId, attrName, attrValue) {
	var e = document.getElementById(compId);
	if (!e) // Component removed or not visible (e.g. on inactive tab of TabPanel)
		return;
		
	var attrNames = attrName.split('.')
	for(var i=0;i<attrNames.length-1;i++) {
	   e = e[attrNames[i]];
	   if (!e) {
	       return;
	   }
	}
	
	e[attrNames[attrNames.length-1]]=attrValue;
	
}

function executeCompFunc(compId, attrName, ...args) {
	var e = document.getElementById(compId);
	if (!e || !e.jsObject || !e.jsObject[attrName]) // check if function exists
		return;
	
	e.jsObject[attrName](...args);
	
}

// Get selected indices (of an HTML select)
function selIdxs(select) {
	var selected = "";
	
	for (var i = 0; i < select.options.length; i++)
		if(select.options[i].selected)
			selected += i + ",";
	
	return selected;
}

// Get and update switch button value
function sbtnVal(event, onBtnId, offBtnId) {
	var onBtn = document.getElementById(onBtnId);
	var offBtn = document.getElementById(offBtnId);
	
	if (onBtn == null)
		return false;
	
	var value = onBtn == document.elementFromPoint(event.clientX, event.clientY);
	if (value) {
		onBtn.className = "gwu-SwitchButton-On-Active";
		offBtn.className = "gwu-SwitchButton-Off-Inactive";
	} else {
		onBtn.className = "gwu-SwitchButton-On-Inactive";
		offBtn.className = "gwu-SwitchButton-Off-Active";
	}
	
	return value;
}

function focusComp(compId) {
	if (compId != null) {
		var e = document.getElementById(compId);
		if (e && e.value) {// Else component removed or not visible (e.g. on inactive tab of TabPanel)
			e.focus();
			e.selectionStart = e.selectionEnd = e.value.length;
		}
	}
}

function scrollDownComp(compId,top) {
	if (compId != null) {
		var e = document.getElementById(compId);
		if (e) {// Else component removed or not visible (e.g. on inactive tab of TabPanel)
			if (top == -1) {
				e.scrollTop = e.scrollHeight;
			} else {
				e.scrollTop = top;
			}
		}
	}
}

function addonload(func) {
	var oldonload = window.onload;
	if (typeof window.onload != 'function') {
		window.onload = func;
	} else {
		window.onload = function() {
			if (oldonload)
				oldonload();
			func();
		}
	}
}

function addonbeforeunload(func) {
	var oldonbeforeunload = window.onbeforeunload;
	if (typeof window.onbeforeunload != 'function') {
		window.onbeforeunload = func;
	} else {
		window.onbeforeunload = function() {
			if (oldonbeforeunload)
				oldonbeforeunload();
			func();
		}
	}
}

var timers = new Object();

function setupTimer(compId, js, timeout, repeat, active, reset) {
	var timer = timers[compId];
	
	if (timer != null) {
		var changed = timer.js != js || timer.timeout != timeout || timer.repeat != repeat || timer.reset != reset;
		if (!active || changed) {
			if (timer.repeat)
				clearInterval(timer.id);
			else
				clearTimeout(timer.id);
			timers[compId] = null;
		}
		if (!changed)
			return;
	}
	if (!active)
		return;
	
	// Create new timer
	timers[compId] = timer = new Object();
	timer.js = js;
	timer.timeout = timeout;
	timer.repeat = repeat;
	timer.reset = reset;
	
	// Start the timer
	if (timer.repeat)
		timer.id = setInterval(js, timeout);
	else
		timer.id = setTimeout(js, timeout);
}

function checkSession(compId) {
	var e = document.getElementById(compId);
	if (!e) // Component removed or not visible (e.g. on inactive tab of TabPanel)
		return;
	
	var xhr = createXmlHttp();
	
	xhr.onreadystatechange = function() {
		if (xhr.readyState == 4 && xhr.status == 200) {
			var timeoutSec = parseFloat(xhr.responseText);
			if (timeoutSec < 60)
				e.classList.add("gwu-SessMonitor-Expired");
			else
				e.classList.remove("gwu-SessMonitor-Expired");
			var cnvtr = window[e.getAttribute("gwuJsFuncName")];
			e.children[0].innerText = typeof cnvtr === 'function' ? cnvtr(timeoutSec) : convertSessTimeout(timeoutSec);
		}
	}
	
	xhr.open("GET", _pathSessCheck, false); // synch call (else we can't catch connection error)
	try {
		xhr.send();
		e.classList.remove("gwu-SessMonitor-Error");
	} catch (err) {
		e.classList.add("gwu-SessMonitor-Error");
		e.children[0].innerText = "CONN ERR";
	}
}

function convertSessTimeout(sec) {
	if (sec <= 0)
		return "Expired!";
	else if (sec < 60)
			return "<1 min";
	else
		return "~" + Math.round(sec / 60) + " min";
}

// INITIALIZATION

addonload(function() {
	focusComp(_focCompId);
});
