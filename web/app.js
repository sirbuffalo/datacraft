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
const fileTabs = document.querySelector('#fileTabs');
const addFileButton = document.querySelector('#addFileButton');
const openProjectButton = document.querySelector('#openProjectButton');
const saveProjectButton = document.querySelector('#saveProjectButton');
const projectFileInput = document.querySelector('#projectFileInput');
const saveStatus = document.querySelector('#saveStatus');
const loadExampleButton = document.querySelector('#loadExampleButton');
const renameFileButton = document.querySelector('#renameFileButton');
const deleteFileButton = document.querySelector('#deleteFileButton');

const legacyStorageKey = 'datacraft.project.v1';
const databaseName = 'datacraft';
const projectStore = 'projects';
const currentProjectKey = 'current';

let latestZIP = null;

CodeMirror.defineMode('datacraft', () => {
  const keywords = new Set(['version', 'namespace', 'from', 'import', 'expose', 'def', 'global', 'return', 'if', 'elif', 'else', 'for', 'while', 'break', 'continue', 'in', 'is', 'and', 'or', 'not', 'mod', 'const', 'readonly']);
  const deprecated = new Set(['export']);
  const types = new Set(['int', 'bool', 'str', 'list', 'set', 'entity', 'nbt']);
  const literals = new Set(['True', 'False', 'None']);
  const builtins = new Set(['say', 'len', 'bool', 'str', 'list', 'set', 'is_bool']);
  const methods = new Set(['append', 'insert', 'remove', 'add', 'discard', 'clear']);
  const commandKeywords = new Set([
    'run', 'if', 'unless', 'as', 'at', 'positioned', 'rotated', 'facing',
    'anchored', 'align', 'in', 'on', 'store', 'result', 'success', 'entity',
    'block', 'blocks', 'biome', 'dimension', 'score', 'matches', 'predicate',
  ]);
  const commandLiterals = new Set(['true', 'false']);

  return {
    startState() {
      return { command: false, commandRoot: false, definition: false, namespace: false, member: false, lineStart: true };
    },
    token(stream, state) {
      if (stream.sol()) {
        state.command = false;
        state.commandRoot = false;
        state.definition = false;
        state.namespace = false;
        state.member = false;
		state.lineStart = true;
      }
	  if (stream.eatSpace()) return null;
      if (state.lineStart && stream.peek() === '/') {
		stream.next();
		state.command = true;
		state.commandRoot = true;
		state.lineStart = false;
		return 'command-prefix';
      }
	  state.lineStart = false;

      if (state.command) {
		if (stream.match(/^\$\([A-Za-z_][A-Za-z0-9_]*\)/)) return 'macro';
		if (stream.match(/^\$\{[^}]*\}/)) return 'interpolation';
		if (stream.match(/^@[paresn](?:\[(?:[^\]"']|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')*\])?/)) return 'selector';
		if (stream.match(/^#?[a-z0-9_.-]+:[a-z0-9_./-]+/i)) return 'resource';
		if (stream.match(/^(?:[~^](?:-?\d+(?:\.\d+)?)?|-?\d+(?:\.\d+)?)(?:\.\.(?:-?\d+(?:\.\d+)?)?)?[bBsSlLfFdD]?/)) return 'coordinate';
		if (stream.match(/^(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/)) return 'string';
		if (stream.match(/^[A-Za-z_][A-Za-z0-9_.-]*(?=\s*:)/)) return 'property';
		if (stream.match(/^[A-Za-z_][A-Za-z0-9_.-]*/)) {
		  const word = stream.current();
		  if (state.commandRoot) {
			state.commandRoot = false;
			return 'command-root';
		  }
		  if (commandKeywords.has(word)) return 'command-keyword';
		  if (commandLiterals.has(word)) return 'atom';
		  return 'command-argument';
		}
		if (stream.match(/^[{}\[\](),:=!]+/)) return 'bracket';
		if (stream.match(/^[-+*/%<>|&]+/)) return 'operator';
		stream.next();
		return 'command-argument';
      }

      if (stream.match(/^#.*/)) return 'comment';
	  if (stream.match(/^@[paresn](?:\[(?:[^\]"']|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')*\])?/)) return 'selector';
      if (stream.match(/^(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/)) return 'string';
      if (stream.match(/^\d+/)) return 'number';
      if (stream.match(/^(?:->|==|!=|<=|>=|\+=|-=|\*=|\/=)/)) return 'operator';
      if (stream.match(/^[()\[\]{},:]/)) return 'bracket';
      if (stream.match(/^[+\-*\/%=<>?|&]/)) return 'operator';
      if (stream.match(/^\./)) {
        state.member = true;
        return 'operator';
      }
      if (stream.match(/^[A-Za-z_][A-Za-z0-9_]*/)) {
        const word = stream.current();
		if (state.definition) {
		  state.definition = false;
		  return 'def';
		}
		if (state.namespace) {
		  state.namespace = false;
		  return 'namespace';
		}
		if (state.member) {
		  state.member = false;
		  return methods.has(word) ? 'builtin' : 'property';
		}
		if (deprecated.has(word)) return 'deprecated';
        if (keywords.has(word)) {
		  if (word === 'def') state.definition = true;
		  if (word === 'namespace' || word === 'from') state.namespace = true;
		  return 'keyword';
		}
		if (literals.has(word)) return 'atom';
		if (builtins.has(word) && /^\s*\(/.test(stream.string.slice(stream.pos))) return 'builtin';
		if (types.has(word)) return 'type';
		if (/^\s*\(/.test(stream.string.slice(stream.pos))) return 'function';
        return 'variable';
      }
      stream.next();
      return 'error';
    },
	lineComment: '#',
  };
});

const editor = CodeMirror.fromTextArea(sourceEditor, {
  mode: 'datacraft',
  lineNumbers: true,
  indentUnit: 4,
  tabSize: 4,
  indentWithTabs: false,
});

const sources = new Map([['main.dcraft', editor.getValue()]]);
let activeFile = 'main.dcraft';
let saveTimer = null;
let databasePromise = null;

function openDatabase() {
  if (databasePromise) return databasePromise;
  databasePromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, 1);
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains(projectStore)) request.result.createObjectStore(projectStore);
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error('Could not open IndexedDB.'));
  });
  return databasePromise;
}

async function readIndexedProject() {
  const database = await openDatabase();
  return new Promise((resolve, reject) => {
    const request = database.transaction(projectStore, 'readonly').objectStore(projectStore).get(currentProjectKey);
    request.onsuccess = () => resolve(request.result || null);
    request.onerror = () => reject(request.error);
  });
}

async function writeIndexedProject(project) {
  const database = await openDatabase();
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(projectStore, 'readwrite');
    transaction.objectStore(projectStore).put(project, currentProjectKey);
    transaction.oncomplete = () => resolve();
    transaction.onerror = () => reject(transaction.error);
    transaction.onabort = () => reject(transaction.error || new Error('IndexedDB write was aborted.'));
  });
}

async function restoreProject() {
  let project = await readIndexedProject();
  if (!project) {
    try {
      const legacy = JSON.parse(localStorage.getItem(legacyStorageKey));
      if (legacy?.sources && Object.keys(legacy.sources).length > 0) {
        project = legacy;
        await writeIndexedProject(legacy);
        localStorage.removeItem(legacyStorageKey);
      }
    } catch (_) {
      // A malformed or unavailable legacy save should not prevent the editor loading.
    }
  }
  if (!project?.sources || Object.keys(project.sources).length === 0) return;
  sources.clear();
  for (const entry of Object.entries(project.sources)) sources.set(...entry);
  activeFile = sources.has(project.activeFile) ? project.activeFile : sources.keys().next().value;
  editor.setValue(sources.get(activeFile));
  packName.value = project.packName || 'example';
  description.value = project.description || 'Built with DataCraft';
  packFormat.value = project.packFormat || 48;
}

async function persistProject() {
  saveActiveFile();
  try {
    await writeIndexedProject({
      activeFile,
      sources: Object.fromEntries(sources),
      packName: packName.value,
      description: description.value,
      packFormat: Number(packFormat.value),
    });
    saveStatus.textContent = 'Saved locally';
    saveStatus.dataset.saving = 'false';
  } catch (_) {
    saveStatus.textContent = 'Local save unavailable';
    saveStatus.dataset.saving = 'true';
  }
}

function scheduleSave() {
  saveStatus.textContent = 'Saving…';
  saveStatus.dataset.saving = 'true';
  clearTimeout(saveTimer);
  saveTimer = setTimeout(persistProject, 250);
}

function saveActiveFile() {
  sources.set(activeFile, editor.getValue());
}

function openFile(filename) {
  if (filename === activeFile) return;
  saveActiveFile();
  activeFile = filename;
  editor.setValue(sources.get(filename) || '');
  editor.focus();
  renderFileTabs();
  updateLineCount();
}

function removeFile(filename) {
  if (sources.size === 1) return;
  if (filename === activeFile) {
    const next = [...sources.keys()].find(name => name !== filename);
    sources.delete(filename);
    activeFile = next;
    editor.setValue(sources.get(next));
  } else {
    sources.delete(filename);
  }
  renderFileTabs();
  updateLineCount();
  scheduleSave();
}

function requestDeleteFile(filename = activeFile) {
  if (sources.size === 1) {
    setMessage('error', 'Cannot delete file', 'A project must contain at least one .dcraft source file.');
    return;
  }
  if (!window.confirm(`Delete ${filename} from this project?`)) return;
  removeFile(filename);
  setMessage('success', 'File deleted', `${filename} was removed from the project.`);
}

function renameActiveFile() {
  const requested = window.prompt('Rename source file:', activeFile);
  if (requested === null) return;
  const filename = requested.trim().replace(/\\/g, '/');
  if (!filename || filename.startsWith('/') || filename.includes('../') || !filename.endsWith('.dcraft')) {
    setMessage('error', 'Invalid filename', 'Use a relative filename ending in .dcraft.');
    return;
  }
  if (filename === activeFile) return;
  if (sources.has(filename)) {
    setMessage('error', 'Filename already used', `${filename} already exists in this project.`);
    return;
  }
  saveActiveFile();
  const renamed = new Map();
  for (const [name, source] of sources) renamed.set(name === activeFile ? filename : name, source);
  sources.clear();
  for (const entry of renamed) sources.set(...entry);
  const previous = activeFile;
  activeFile = filename;
  renderFileTabs();
  persistProject();
  setMessage('success', 'File renamed', `${previous} is now ${filename}.`);
}

function renderFileTabs() {
  fileTabs.replaceChildren();
  deleteFileButton.disabled = sources.size === 1;
  for (const filename of sources.keys()) {
    const tab = document.createElement('button');
    tab.type = 'button';
    tab.className = 'file-tab';
    tab.dataset.active = String(filename === activeFile);
    tab.addEventListener('click', () => openFile(filename));
    const dot = document.createElement('span');
    dot.className = 'file-dot';
    const label = document.createElement('span');
    label.textContent = filename;
    tab.append(dot, label);
    if (sources.size > 1) {
      const close = document.createElement('span');
      close.className = 'file-close';
      close.textContent = '×';
      close.title = `Remove ${filename}`;
      close.addEventListener('click', event => {
        event.stopPropagation();
        requestDeleteFile(filename);
      });
      tab.append(close);
    }
    fileTabs.append(tab);
  }
}

addFileButton.addEventListener('click', () => {
  saveActiveFile();
  let number = 1;
  while (sources.has(`module${number}.dcraft`)) number += 1;
  const filename = `module${number}.dcraft`;
  sources.set(filename, `namespace module${number}\n\ndef example():\n    say("module${number}")\n`);
  openFile(filename);
  scheduleSave();
});
renameFileButton.addEventListener('click', renameActiveFile);
deleteFileButton.addEventListener('click', () => requestDeleteFile());

function projectConfig() {
  const name = packName.value.trim() || 'example';
  const quote = value => JSON.stringify(value);
  return `[pack]\nname = ${quote(name)}\ndescription = ${quote(description.value)}\nformat = ${Number(packFormat.value) || 48}\n\n[build]\nsource = "src"\noutput = "build/${name}.zip"\n`;
}

function loadExample() {
  if (!window.confirm('Replace the current editor project with the example? Your current project remains in IndexedDB until you confirm.')) return;
  const example = `version 2
namespace example

expose def add(a: int, b: int) -> int:
    total: int = a + b
    return total

expose def load():
    values: list[int] = [2, 3, 5]
    result: int = add(values[0], values[1])
    profile: nbt = {"name": "Alex", "score": result, "flags": [True, False]}
    if result == 5:
        say("DataCraft example: ", profile)
    else:
        say("Unexpected result: ", result)
`;
  sources.clear();
  sources.set('main.dcraft', example);
  activeFile = 'main.dcraft';
  packName.value = 'example';
  description.value = 'DataCraft example project';
  packFormat.value = 48;
  editor.setValue(example);
  renderFileTabs();
  updateLineCount();
  latestZIP = null;
  downloadButton.disabled = true;
  fileList.replaceChildren();
  outputSummary.textContent = 'Compile to inspect generated files.';
  persistProject();
  setMessage('success', 'Example loaded', 'The example project is ready to compile.');
  editor.focus();
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function saveProjectZIP() {
  persistProject();
  const files = { 'datacraft.toml': projectConfig() };
  for (const [name, source] of sources) {
    const safeName = name.replace(/\\/g, '/').replace(/^(\.\.\/|\/)+/g, '');
    files[`src/${safeName}`] = source;
  }
  const bytes = DataCraftZIP.create(files);
  downloadBlob(new Blob([bytes], { type: 'application/zip' }), `${packName.value.trim() || 'datacraft-project'}-source.zip`);
  setMessage('success', 'Project saved', 'Downloaded source files and datacraft.toml as a ZIP.');
}

function parseProjectConfig(text) {
  const readString = key => text.match(new RegExp(`^\\s*${key}\\s*=\\s*"((?:[^"\\\\]|\\\\.)*)"`, 'm'))?.[1]
    ?.replace(/\\"/g, '"').replace(/\\\\/g, '\\');
  const format = Number(text.match(/^\s*format\s*=\s*(\d+)/m)?.[1]);
  return { name: readString('name'), description: readString('description'), format };
}

async function openProjectZIP(file) {
  const archive = await DataCraftZIP.open(await file.arrayBuffer());
  const names = [...archive.keys()];
  const configName = names.find(name => name === 'datacraft.toml') || names.find(name => name.endsWith('/datacraft.toml'));
  const root = configName ? configName.slice(0, -'datacraft.toml'.length) : '';
  const sourcePrefix = `${root}src/`;
  const openedSources = new Map();
  for (const [name, bytes] of archive) {
    if (!name.endsWith('.dcraft')) continue;
    const relative = name.startsWith(sourcePrefix) ? name.slice(sourcePrefix.length) : name.slice(root.length);
    if (relative && !relative.startsWith('../')) openedSources.set(relative, DataCraftZIP.decode(bytes));
  }
  if (openedSources.size === 0) throw new Error('The ZIP does not contain any .dcraft source files.');
  if (configName) {
    const config = parseProjectConfig(DataCraftZIP.decode(archive.get(configName)));
    if (config.name) packName.value = config.name;
    if (config.description !== undefined) description.value = config.description;
    if (config.format) packFormat.value = config.format;
  }
  sources.clear();
  for (const entry of openedSources) sources.set(...entry);
  activeFile = sources.keys().next().value;
  editor.setValue(sources.get(activeFile));
  renderFileTabs();
  updateLineCount();
  persistProject();
  latestZIP = null;
  downloadButton.disabled = true;
  fileList.replaceChildren();
  outputSummary.textContent = 'Compile to inspect generated files.';
  setMessage('success', 'Project opened', `Loaded ${sources.size} source file${sources.size === 1 ? '' : 's'} from ${file.name}.`);
}

saveProjectButton.addEventListener('click', saveProjectZIP);
loadExampleButton.addEventListener('click', loadExample);
openProjectButton.addEventListener('click', () => projectFileInput.click());
projectFileInput.addEventListener('change', async () => {
  const file = projectFileInput.files[0];
  projectFileInput.value = '';
  if (!file) return;
  try {
    await openProjectZIP(file);
  } catch (error) {
    setMessage('error', 'Could not open project', error.message);
  }
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
  if (typeof window.datacraftCompile !== 'function') return;
  compileButton.disabled = true;
  downloadButton.disabled = true;
  latestZIP = null;
  fileList.replaceChildren();
  setMessage('idle', 'Compiling…', 'Parsing source and assembling the data pack.');

  requestAnimationFrame(() => {
	 saveActiveFile();
	 const projectSources = Object.fromEntries(sources);
	 const usesProject = sources.size > 1 || [...sources.values()].some(source => /^\s*from\s+/m.test(source));
    const raw = window.datacraftCompile(JSON.stringify({
      source: editor.getValue(),
	  sources: usesProject ? projectSources : undefined,
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
  downloadBlob(latestZIP, `${packName.value.trim() || 'datapack'}.zip`);
});

editor.on('change', () => {
  updateLineCount();
  sources.set(activeFile, editor.getValue());
  scheduleSave();
});
[packName, description, packFormat].forEach(input => input.addEventListener('input', scheduleSave));
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

async function initializeEditor() {
  try {
    await restoreProject();
  } catch (_) {
    saveStatus.textContent = 'Local save unavailable';
    saveStatus.dataset.saving = 'true';
  }
  renderFileTabs();
  updateLineCount();
  startCompiler();
}

initializeEditor();
