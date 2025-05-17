package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "path/filepath"

    "github.com/mdp/qrterminal/v3"
)

const (
    uploadDir  = "./uploads"
    staticDir  = "./static"
    serverPort = "8080"
)

func main() {
    // 1) Ensure uploads directory exists
    if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
        if err := os.Mkdir(uploadDir, 0755); err != nil {
            fmt.Fprintf(os.Stderr, "Error creating upload directory: %v\n", err)
            os.Exit(1)
        }
    }

    // 2) Routes
    http.Handle("/", http.FileServer(http.Dir(staticDir)))       // single‑page UI
    http.HandleFunc("/upload", uploadHandler)                    // handle uploads
    http.Handle("/files/", http.StripPrefix("/files/",           // serve saved files
        http.FileServer(http.Dir(uploadDir))))
    http.HandleFunc("/api/files", filesAPIHandler)               // JSON file list

    // 3) Print access URLs + QR code
    ip := getLocalIP()
    fmt.Println("Server started at:")
    fmt.Printf("→ http://localhost:%s/\n", serverPort)
    if ip != "" {
        url := fmt.Sprintf("http://%s:%s/", ip, serverPort)
        fmt.Printf("→ %s\n", url)
        qrterminal.Generate(url, qrterminal.L, os.Stdout)
    }

    // 4) Start HTTP server
    if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
        fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
        os.Exit(1)
    }
}

// uploadHandler saves an incoming file then redirects back to “/”
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error reading file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    dstPath := filepath.Join(uploadDir, filepath.Base(header.Filename))
    dst, err := os.Create(dstPath)
    if err != nil {
        http.Error(w, "Unable to create file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    if _, err := io.Copy(dst, file); err != nil {
        http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

// filesAPIHandler returns the list of uploaded filenames as JSON
func filesAPIHandler(w http.ResponseWriter, r *http.Request) {
    entries, err := os.ReadDir(uploadDir)
    if err != nil {
        http.Error(w, "Unable to read uploads: "+err.Error(), http.StatusInternalServerError)
        return
    }
    names := make([]string, 0, len(entries))
    for _, e := range entries {
        if !e.IsDir() {
            names = append(names, e.Name())
        }
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(names)
}

// getLocalIP finds the first non-loopback IPv4 address
func getLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return ""
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok &&
            !ipnet.IP.IsLoopback() &&
            ipnet.IP.To4() != nil {
            return ipnet.IP.String()
        }
    }
    return ""
}
