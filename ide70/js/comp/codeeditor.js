class EditorBlock {
	constructor(parent) {
    	this.parentElement = parent;
		parent.jsObject = this;
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
		bindings.set("yaml","yaml");
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
				editorNode.dataEditor.commands.addCommand({
				    name: "save",
				    bindKey: {win: "Ctrl-S", mac: "Command-Option-S"},
				    exec: function(editor) {
				        editorBlock.save(editorNode);
				    }
				});
    		}
  		};
  		xhttp.open("GET", "/app/_fs/" + editorNode.key + "?dummy="+Math.random(), true);
  		xhttp.send();
	}
	
	save(editorNode) {
		var editorBlock = this;
	    var xhttp = new XMLHttpRequest();
  		xhttp.onreadystatechange = function() {
    		if (this.readyState == 4 && this.status == 200) {
     			editorBlock.markClean(editorNode);
    		}
  		};
  		  		
  		xhttp.open("POST", "/app/_save/" + editorNode.key, true);
  		xhttp.responseType = "blob";
  		xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
  		xhttp.send("content=" + encodeURIComponent(editorNode.dataEditor.getValue()));
    }

	
}