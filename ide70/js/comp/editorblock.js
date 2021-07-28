class EditorBlock {
	constructor(parent, titleElement) {
    	this.parentElement = parent;
		parent.jsObject = this;
		this.titleElement = titleElement;
    	ace.require('ace/ext/language_tools');
    	this.activeEditors = new Map();
    	this.selectedEditorKey = "";
    	this.fileExtensionBindings = this.defaultFileExtensionBindings();
		var editorBlock = this;
    	this.designWordCompleter = {
       	    getCompletions: function(editor, session, pos, prefix, callback) {
       	     	var xhttp = new XMLHttpRequest();  		
    	  		xhttp.open("POST", _pathApp+"_codeComplete/"+editorBlock.selectedEditorKey, true);
    	  		xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");  
    	  		
    	  		xhttp.onreadystatechange = function() {
    	    		if (this.readyState == 4 && this.status == 200) {
    	    			var completions = JSON.parse(this.responseText);
    	    			callback(null, completions);
    	    		}
    	  		};
    	  		
    	  		var edStr = editor.getValue();
    	  		xhttp.send("row=" + pos.row + "&col=" + pos.column+ "&content=" + encodeURIComponent(edStr));
    
       	    },
        	identifierRegexps: [ /[a-zA-Z_0-9\-\u00A2-\uFFFF]/ ]
       	};
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
		this.displayCleanStatus(editorNode);
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
		editorNode.title = key.substr(6);
		editorNode.rootElement = document.createElement('div');
		editorNode.titleElement = this.titleElement;
		editorNode.editorElement = document.createElement('pre');
		editorNode.rootElement.style.display = "none";
		editorNode.elementId = "ed_"+key;
		editorNode.editorElement.id = editorNode.elementId;
		editorNode.rootElement.appendChild(editorNode.editorElement);
		this.parentElement.appendChild(editorNode.rootElement);
		
		editorNode.dataEditor = ace.edit(editorNode.elementId);
		editorNode.dataEditor.setTheme("ace/theme/twilight");
		var fileExtension = key.split('.').pop();
		var mode = this.fileExtensionBindings.get(fileExtension);
		if (mode) {
			editorNode.dataEditor.session.setMode("ace/mode/" + mode);
		}
		editorNode.dataEditor.setOptions({
            enableBasicAutocompletion: true
        });
        editorNode.dataEditor.completers = [this.designWordCompleter];
		
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
		}
		editorNode.titleElement.innerHTML = (isClean ? "" : "*") + editorNode.title;
	}
	
	load(editorNode){
		var editorBlock = this;
		var xhttp = new XMLHttpRequest();
  		xhttp.onreadystatechange = function() {
    		if (this.readyState == 4 && this.status == 200) {
    			editorNode.dataEditor.setValue(this.responseText, -1);
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
				var nrLines = editorNode.dataEditor.session.getLength();
				if(nrLines > 80) {
				    editorNode.dataEditor.setFontSize("12pt");
				} else if(nrLines > 40) {
				    editorNode.dataEditor.setFontSize("13pt");
				}
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
  		  		
  		xhttp.open("POST", _pathApp+"_save/" + editorNode.key, true);
  		xhttp.responseType = "blob";
  		xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
  		xhttp.send("content=" + encodeURIComponent(editorNode.dataEditor.getValue()));
    }
}