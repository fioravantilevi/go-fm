package main

import (
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/webview/webview_go"
)

var (
	mainTmpl     *template.Template
	fmTmpl       *template.Template
	staticServer *http.Server
)

const mainLayout = `
<!DOCTYPE html>
<html>
<head>
<title>Go-Fm</title>
<style>
body { margin: 0; background: #232629; color: #eff0f1; font-family: monospace; display: flex; height: 100vh; overflow: hidden; }
#sidebar { width: 0; transition: 0.3s ease; background: #1b1e20; overflow: hidden; }
#sidebar.open { width: 400px; border-right: 2px solid #3daee9; }
.workspace { flex-grow: 1; display: flex; flex-direction: column; }
.header { padding: 10px; background: #31363b; display: flex; align-items: center; gap: 15px; border-bottom: 1px solid #3daee9; }
.editor-container { flex-grow: 1; position: relative; display: flex; flex-direction: column; }
textarea, #preview-frame { flex-grow: 1; border: none; outline: none; width: 100%; height: 100%; }
textarea { background: transparent; color: #eff0f1; padding: 20px; font-size: 16px; line-height: 1.5; resize: none; }
#preview-frame { background: #fff; display: none; }
button { background: #3daee9; color: white; border: none; padding: 5px 12px; cursor: pointer; font-weight: bold; font-family: monospace; }
.btn-save { background: #27ae60; }
.btn-lsp { background: #8e44ad; }
.btn-test { background: #fdbc4b; color: #232629; margin-left: auto; }
#file-path { color: #fdbc4b; font-size: 12px; }
</style>
</head>
<body>
<div id="sidebar"><iframe src="/fm" style="width:400px; height:100%; border:none;"></iframe></div>
<div class="workspace">
<div class="header">
<button onclick="toggleFM()">☰ MENU</button>
<span id="file-path">No file open</span>
<button class="btn-lsp" onclick="runLSP()">Go LSP Test</button>
<button class="btn-test" onclick="toggleTestServer()" id="test-btn">START TEST SERVER</button>
<button class="btn-save" onclick="saveFile()">SAVE</button>
</div>
<div class="editor-container">
<textarea id="editor" spellcheck="false"></textarea>
<iframe id="preview-frame"></iframe>
</div>
</div>
<script>
let serverRunning = false;
function toggleFM() { document.getElementById('sidebar').classList.toggle('open'); }

window.addEventListener('message', e => {
if (e.data.type === 'open') openFile(e.data.path);
});

async function openFile(path) {
const res = await fetch('/read?path=' + encodeURIComponent(path));
document.getElementById('editor').value = await res.text();
document.getElementById('file-path').innerText = path;
}

async function runLSP() {
const path = document.getElementById('file-path').innerText;
if (path === "No file open") return;
const res = await fetch('/check?path=' + encodeURIComponent(path));
const report = await res.text();
alert(report === "" ? "Logic is clean!" : "LSP Report:\n\n" + report);
}

async function toggleTestServer() {
const frame = document.getElementById('preview-frame');
const editor = document.getElementById('editor');
const btn = document.getElementById('test-btn');
const path = document.getElementById('file-path').innerText;

if (!serverRunning) {
	await fetch('/start-server');
	editor.style.display = 'none';
	frame.style.display = 'block';
	const filename = path.split('/').pop();
	frame.src = "http://127.0.0.1:8081/" + filename;
	btn.innerText = "STOP TEST SERVER";
	btn.style.background = "#ed1515";
	btn.style.color = "white";
	serverRunning = true;
	} else {
		await fetch('/stop-server');
		editor.style.display = 'block';
		frame.style.display = 'none';
		btn.innerText = "START TEST SERVER";
		btn.style.background = "#fdbc4b";
		btn.style.color = "#232629";
		serverRunning = false;
		}
		}

		async function saveFile() {
		const path = document.getElementById('file-path').innerText;
		const content = document.getElementById('editor').value;
		if (path === "No file open") return;
		await fetch('/save', {
		method: 'POST',
headers: {'Content-Type': 'application/x-www-form-urlencoded'},
body: 'path=' + encodeURIComponent(path) + '&content=' + encodeURIComponent(content)
});
if (serverRunning) document.getElementById('preview-frame').contentWindow.location.reload();
}

document.addEventListener('keydown', e => {
if (e.ctrlKey && e.key === 's') { e.preventDefault(); saveFile(); }
if (e.ctrlKey && e.key === 'b') { e.preventDefault(); toggleFM(); }
});
</script>
</body>
</html>`

const fmLayout = `
<!DOCTYPE html>
<html>
<head>
<style>
body { font-family: monospace; background: #1b1e20; color: #eff0f1; padding: 15px; margin: 0; display: flex; flex-direction: column; height: 100vh; box-sizing: border-box; }
.nav-bar { margin-bottom: 15px; border-bottom: 1px solid #3daee9; padding-bottom: 10px; display: flex; align-items: center; gap: 8px; }
.nav-btn { color: #fdbc4b; text-decoration: none; font-weight: bold; font-size: 12px; }
.path-input { background: #232629; color: #eff0f1; border: 1px solid #4d5052; padding: 3px 6px; font-family: monospace; font-size: 11px; flex-grow: 1; outline: none; }
.list { flex-grow: 1; overflow-y: auto; }
.file-row { display: flex; justify-content: space-between; padding: 4px 0; border-bottom: 1px solid #31363b; }
.name-link { color: #3daee9; text-decoration: none; cursor: pointer; flex-grow: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.is-file { color: #eff0f1; }
.btn-del { color: #ed1515; text-decoration: none; font-size: 0.8em; }
.create-bar { margin-top: 20px; padding-top: 15px; border-top: 1px solid #4d5052; display: flex; flex-direction: column; gap: 10px; }
input { background: #232629; color: #fdbc4b; border: 1px solid #4d5052; padding: 5px; outline: none; }
button { border: none; padding: 5px; cursor: pointer; color: white; font-weight: bold; }
</style>
</head>
<body>
<div class="nav-bar">
<a href="/home" class="nav-btn">[H]</a>
<a href="/cd?path=.." class="nav-btn">[U]</a>
<form action="/cd" method="GET" style="display:contents;">
<input type="text" name="path" class="path-input" value="{{.Path}}" spellcheck="false" autocomplete="off">
</form>
</div>
<div class="list">
{{range .Files}}
<div class="file-row">
{{if .IsDir}}
<a href="/cd?path={{.Name}}" class="name-link">{{.Name}}/</a>
{{else}}
<span class="name-link is-file" onclick="parent.postMessage({type:'open', path:'{{$.Path}}/{{.Name}}'}, '*')">{{.Name}}</span>
<a href="/delete?name={{.Name}}" class="btn-del" onclick="return confirm('Delete?')">del</a>
{{end}}
</div>
{{end}}
</div>
<div class="create-bar">
<form action="/newfolder" method="GET" style="display:flex; gap:5px;">
<input type="text" name="name" placeholder="Folder..." required style="flex-grow:1;">
<button type="submit" style="background: #3daee9;">+DIR</button>
</form>
<form action="/newfile" method="GET" style="display:flex; gap:5px;">
<input type="text" name="name" placeholder="File..." required style="flex-grow:1;">
<button type="submit" style="background: #27ae60;">+FILE</button>
</form>
</div>
</body>
</html>`

type PageData struct {
	Path  string
	Files []os.DirEntry
}

func main() {
	os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	os.Setenv("GDK_BACKEND", "x11") // Keep this if you still need it for Wayland
	mainTmpl = template.Must(template.New("main").Parse(mainLayout))
	fmTmpl = template.Must(template.New("fm").Parse(fmLayout))

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { mainTmpl.Execute(w, nil) })

	mux.HandleFunc("/fm", func(w http.ResponseWriter, r *http.Request) {
		cwd, _ := os.Getwd()
		files, _ := os.ReadDir(".")
		fmTmpl.Execute(w, PageData{Path: cwd, Files: files})
	})

	mux.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		home, _ := os.UserHomeDir()
		os.Chdir(home)
		http.Redirect(w, r, "/fm", http.StatusSeeOther)
	})

	mux.HandleFunc("/cd", func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("path")
		if target != "" {
			os.Chdir(target)
		}
		http.Redirect(w, r, "/fm", http.StatusSeeOther)
	})

	mux.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		content, _ := os.ReadFile(path)
		w.Write(content)
	})

	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		os.WriteFile(r.FormValue("path"), []byte(r.FormValue("content")), 0644)
	})

	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		cmd := exec.Command("go", "vet", path)
		out, _ := cmd.CombinedOutput()
		w.Write(out)
	})

	mux.HandleFunc("/start-server", func(w http.ResponseWriter, r *http.Request) {
		if staticServer != nil {
			return
		}
		staticServer = &http.Server{Addr: "127.0.0.1:8081", Handler: http.FileServer(http.Dir("."))}
		go staticServer.ListenAndServe()
	})

	mux.HandleFunc("/stop-server", func(w http.ResponseWriter, r *http.Request) {
		if staticServer != nil {
			staticServer.Close()
			staticServer = nil
		}
	})

	mux.HandleFunc("/newfolder", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name != "" {
			os.Mkdir(name, 0755)
		}
		http.Redirect(w, r, "/fm", http.StatusSeeOther)
	})

	mux.HandleFunc("/newfile", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name != "" {
			f, _ := os.Create(name)
			f.Close()
		}
		http.Redirect(w, r, "/fm", http.StatusSeeOther)
	})

	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name != "" {
			os.RemoveAll(name)
		}
		http.Redirect(w, r, "/fm", http.StatusSeeOther)
	})

	// 1. Bind to the port FIRST. This guarantees the server is ready.
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		server.Serve(listener)
	}()

	// 2. Create and Run the Native Webview
	debug := true // Set to true so you can right-click -> Inspect Element
	w := webview.New(debug)
	defer w.Destroy()

	w.SetTitle("Go-FM Text Editor")
	w.SetSize(1200, 800, webview.HintNone)
	w.Navigate("http://127.0.0.1:8080/")

	// 3. This blocks the program from exiting until you close the window
	w.Run()
}
