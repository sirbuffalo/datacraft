const runtimeStatus = document.querySelector('#runtimeStatus');
const compileButton = document.querySelector('#compileButton');
const downloadButton = document.querySelector('#downloadButton');
const sourceEditor = document.querySelector('#sourceEditor');
const packName = document.querySelector('#packName');
const description = document.querySelector('#description');
const packFormat = document.querySelector('#packFormat');
const message = document.querySelector('#message');
const fileList = document.querySelector('#fileList');
const outputSummary = document.querySelector('#outputSummary');
const lineCount = document.querySelector('#lineCount');

let latestZIP = null;

CodeMirror.defineMode('mccomp', () => {
  const keywords = new Set(['namespace', 'export', 'def', 'global', 'return', 'if', 'elif', 'else', 'for', 'while', 'break', 'continue', 'in', 'is', 'and', 'or', 'not', 'mod']);
  const atoms = new Set(['True', 'False', 'int', 'bool', 'str', 'list', 'entity']);
  const commandAtoms = new Set(['true', 'false', 'run', 'if', 'unless', 'as', 'at', 'positioned', 'rotated', 'facing', 'anchored', 'in', 'on', 'store', 'result', 'success']);

  return {
    startState() {
      return { command: false, commandRoot: false };
    },
    token(stream, state) {
      if (stream.sol()) {
        state.command = false;
        state.commandRoot = false;
      }
      if (stream.sol() && stream.peek() === '/') {
		stream.next();
		state.command = true;
		state.commandRoot = true;
		return 'command-prefix';
      }
      if (stream.eatSpace()) return null;

      if (state.command) {
		if (stream.match(/^\$\([A-Za-z_][A-Za-z0-9_]*\)/)) return 'macro';
		if (stream.match(/^@[pares](?:\[[^\]]*\])?/)) return 'selector';
		if (stream.match(/^#?[a-z0-9_.-]+:[a-z0-9_./-]+/)) return 'resource';
		if (stream.match(/^(?:[~^](?:-?\d+(?:\.\d+)?)?|-?\d+(?:\.\d+)?)(?:\.\.(?:-?\d+(?:\.\d+)?)?)?/)) return 'coordinate';
		if (stream.match(/^(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/)) return 'string';
		if (stream.match(/^[A-Za-z_][A-Za-z0-9_.-]*(?=\s*:)/)) return 'property';
		if (stream.match(/^[A-Za-z_][A-Za-z0-9_.-]*/)) {
		  const word = stream.current();
		  if (state.commandRoot) {
			state.commandRoot = false;
			return 'command-root';
		  }
		  return commandAtoms.has(word) ? 'command-keyword' : 'command-argument';
		}
		if (stream.match(/^[{}\[\](),:=!]+/)) return 'bracket';
		if (stream.match(/^[-+*/%<>]+/)) return 'operator';
		stream.next();
		return 'command-argument';
      }

      if (stream.match(/^#.*/)) return 'comment';
	  if (stream.match(/^@[pares](?:\[[^\]]*\])?/)) return 'selector';
      if (stream.match(/^(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/)) return 'string';
      if (stream.match(/^\d+/)) return 'number';
      if (stream.match(/^(?:==|!=|<=|>=|\+=|-=|\*=|\/=|[+\-*\/%=<>:(),\[\]])/)) return 'operator';
      if (stream.match(/^[A-Za-z_][A-Za-z0-9_]*/)) {
        const word = stream.current();
        if (keywords.has(word)) return 'keyword';
        if (atoms.has(word)) return 'atom';
        return 'variable';
      }
      stream.next();
      return 'error';
    },
	lineComment: '#',
  };
});

const editor = CodeMirror.fromTextArea(sourceEditor, {
  mode: 'mccomp',
  lineNumbers: true,
  indentUnit: 4,
  tabSize: 4,
  indentWithTabs: false,
});

function updateLineCount() {
  const count = editor.lineCount();
  lineCount.textContent = `${count} ${count === 1 ? 'line' : 'lines'}`;
}

function setMessage(kind, title, detail) {
  message.dataset.kind = kind;
  message.replaceChildren();
  const strong = document.createElement('strong');
  const span = document.createElement('span');
  strong.textContent = title;
  span.textContent = detail;
  message.append(strong, span);
}

function decodeBase64(value) {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes;
}

function compile() {
  if (typeof window.mccompCompile !== 'function') return;
  compileButton.disabled = true;
  downloadButton.disabled = true;
  latestZIP = null;
  fileList.replaceChildren();
  setMessage('idle', 'Compiling…', 'Parsing source and assembling the data pack.');

  requestAnimationFrame(() => {
    const raw = window.mccompCompile(JSON.stringify({
      source: editor.getValue(),
      packName: packName.value.trim(),
      description: description.value.trim(),
      packFormat: Number(packFormat.value),
    }));
    const result = JSON.parse(raw);
    compileButton.disabled = false;
    if (!result.ok) {
      outputSummary.textContent = 'Build failed.';
      setMessage('error', 'Compiler error', result.error || 'Unknown compiler error');
      return;
    }

    latestZIP = new Blob([decodeBase64(result.zip)], { type: 'application/zip' });
    for (const filename of result.files) {
      const item = document.createElement('div');
      item.className = 'file-item';
      item.textContent = filename;
      fileList.append(item);
    }
    outputSummary.textContent = `${result.files.length} generated files.`;
    setMessage('success', 'Build succeeded', 'Your data pack is ready to download.');
    downloadButton.disabled = false;
  });
}

downloadButton.addEventListener('click', () => {
  if (!latestZIP) return;
  const url = URL.createObjectURL(latestZIP);
  const link = document.createElement('a');
  link.href = url;
  link.download = `${packName.value.trim() || 'datapack'}.zip`;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
});

editor.on('change', updateLineCount);
editor.setOption('extraKeys', {
  Tab(instance) {
    instance.replaceSelection('    ', 'end');
  },
  'Cmd-Enter': compile,
  'Ctrl-Enter': compile,
});
compileButton.addEventListener('click', compile);

async function startCompiler() {
  try {
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(fetch('compiler.wasm'), go.importObject);
    go.run(result.instance);
    runtimeStatus.dataset.state = 'ready';
    runtimeStatus.lastElementChild.textContent = 'Compiler ready';
    compileButton.disabled = false;
  } catch (error) {
    runtimeStatus.dataset.state = 'error';
    runtimeStatus.lastElementChild.textContent = 'Compiler unavailable';
    setMessage('error', 'Could not load WebAssembly', error.message);
  }
}

updateLineCount();
startCompiler();
