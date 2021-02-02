//import * as ace  from './ace.js';

class _Component {
	constructor(parent) {
    	this.parentElement = parent;
  	}
}

class FileTree extends _Component {
	constructor(parent, edBlock) {
    	super(parent);
    	this.editorBlock = edBlock;
    	this.fileNames = new Map();
  	}

	populate(treeJSON) {
		var tree = JSON.parse(treeJSON);

		var ul = document.createElement('ul');
		for (const val of tree) {
			this.populateItem(val, ul, "")
		}
		this.parentElement.innerHTML = '';
		this.parentElement.appendChild(ul);
	}
	
	populateItem(value, parent, path) {
		var li = document.createElement('li');
		var img = document.createElement('i');
		var span = document.createElement('span');
		li.appendChild(img)
		li.appendChild(span)
		if(Array.isArray(value)) {
			img.className = "fas fa-folder fa-fw";
			span.innerHTML = value[0];
			span.className = "fs-dir";
			this.addToggler(span);
			var ul = document.createElement('ul');
			for (const val of value.slice(1)) {
				this.populateItem(val, ul, path + "/" + value[0]);
			}
			li.appendChild(ul)
		} else {
			img.className = "far fa-file-alt fa-fw";
			span.innerHTML = value;
			this.addOpenFile(span, path + "/" + value);
		}
		parent.appendChild(li);
	}
	
	addToggler(element) {
		element.addEventListener("click", function() {
			this.classList.toggle("open");
		});
	}
	
	addOpenFile(element, key) {
		var fileSelect = this;
		element.addEventListener("dblclick", function() {
			fileSelect.editorBlock.select(key);
		});
	}

	fetch() {
    		var xhttp = new XMLHttpRequest();
		var fileSelect = this;

  		xhttp.onreadystatechange = function() {
	    		if (this.readyState == 4 && this.status == 200) {
	    			fileSelect.populate(this.responseText);
	    		}
  		};
  		
    		xhttp.open("GET", "/list", true);
  		xhttp.send();
	}
}

class EditorBlock extends _Component {
	constructor(parent) {
    	super(parent);
    	ace.require('ace/ext/language_tools');
    	this.activeEditors = new Map();
    	this.selectedEditorKey = "";
    	this.fileExtensionBindings = this.defaultFileExtensionBindings();
  	}
	
	defaultFileExtensionBindings() {
		var bindings = new Map();
		bindings.set("css","css");
		bindings.set("html","html");
		bindings.set("java","java");
		bindings.set("js","javascript");
		bindings.set("json","json");
		return bindings;
	}
	
	select(key) {
		if(key == this.selectedEditorKey) {
			return;
		}
		var editorNode = this.activeEditors.get(key);
		if(!editorNode) {
			editorNode =this.newEditor(key);
			this.load(editorNode);
		}
		var oldEditorNode = this.activeEditors.get(this.selectedEditorKey);
		this.hide(oldEditorNode);
		this.selectedEditorKey = key;
		this.show(editorNode);
	}
	
	show(editorNode) {
		if(editorNode) {
			editorNode.rootElement.style.display = "block";
			editorNode.dataEditor.focus();
		}
	}
	
	hide(editorNode) {
		if(editorNode) {
			editorNode.rootElement.style.display = "none";
		}
	}
	
	newEditor(key){
		var editorBlock = this;
		var editorNode = {};
		editorNode.key = key;
		editorNode.title = key.substr(1);
		editorNode.rootElement = document.createElement('div');
		editorNode.titleElement = document.createElement('div');
		editorNode.titleElement.className = "title ace_editor ace-twilight";
		editorNode.editorElement = document.createElement('pre');
		editorNode.rootElement.style.display = "none";
		editorNode.elementId = "ed_"+key;
		editorNode.editorElement.id = editorNode.elementId;
		editorNode.rootElement.appendChild(editorNode.titleElement);
		editorNode.rootElement.appendChild(editorNode.editorElement);
		this.parentElement.appendChild(editorNode.rootElement);
		editorNode.dataEditor = ace.edit(editorNode.elementId);
		editorNode.dataEditor.setTheme("ace/theme/twilight");
		var fileExtension = key.split('.').pop();
		var mode = this.fileExtensionBindings.get(fileExtension);
		if (mode) {
			editorNode.dataEditor.session.setMode("ace/mode/" + mode);
		}
		this.activeEditors.set(key, editorNode);
		return editorNode;
	}
	
	markClean(editorNode) {
		editorNode.cleanStatus = false;
		editorNode.dataEditor.session.getUndoManager().reset();
		//editorNode.dataEditor.session.getUndoManager().markClean();
		this.displayCleanStatus(editorNode);
	}
	
	displayCleanStatus(editorNode) {
		var isClean = !editorNode.dataEditor.session.getUndoManager().hasUndo();
		if(editorNode.cleanStatus != isClean) {
			editorNode.cleanStatus = isClean;
			editorNode.titleElement.innerHTML = (isClean ? "" : "*") + editorNode.title;
		}
	}
	
	load(editorNode){
		var editorBlock = this;
		var xhttp = new XMLHttpRequest();
  		xhttp.onreadystatechange = function() {
    		if (this.readyState == 4 && this.status == 200) {
    			editorNode.dataEditor.setValue(this.responseText);
    			editorBlock.markClean(editorNode);
    			editorNode.dataEditor.session.on('change', function() {
    				editorBlock.displayCleanStatus(editorNode);
    			});
    		}
  		};
  		xhttp.open("GET", "/load/" + editorNode.key + "?dummy="+Math.random(), true);
  		xhttp.send();
	}
	
}

class NavMenu extends _Component {
	
	constructor(parent) {
    	super(parent);
    	this.items = [];
    	this.topLevel = true;
  	}
	
}

class NavMenuItem {
	constructor(title, url, icon) {
		this.title = title;
		this.url = url;
		this.icon = icon;
	}
}
