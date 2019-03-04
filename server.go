package main

import (
	"net/http"
	"strings"
	"io"
	"io/ioutil"
	"os"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type PieceInfo struct 
{
	Name string `json:"name"`
	Audio string `json:"audio"`
	PDF string `json:"pdf"`
	Map map[string]uint `json:"map"`
	Credits map[string]string `json:"credits"`
}

func getPieceInfo(path string) (*PieceInfo) {
	if strings.Index(path, ".") != -1  {
		return nil
	}

	jsonraw, err := ioutil.ReadFile("./data/" + path + ".json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	
	jsondata := &PieceInfo{}
	err = json.Unmarshal(jsonraw, jsondata)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)	
		return nil
	}

	return jsondata
}

func getPdfPage(url string, page uint) []byte {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}

	tmp, err := ioutil.TempFile("", "*.pdf")
	tmp.Write(body)
	defer tmp.Close()
	out, err := exec.Command("gs", "-q", "-dSAFER", "-dBATCH", "-dNOPAUSE", "-sDEVICE=pnggray", "-sPageList=" + fmt.Sprint(page), "-sOutputFile=-", tmp.Name()).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	return out
}

func main() {
	http.HandleFunc("/files/", func (w http.ResponseWriter, r *http.Request) {
		if strings.Index(r.URL.Path, "..") != -1 {
			w.WriteHeader(403)
			return
		}
		http.ServeFile(w, r, "." + r.URL.Path)
	})
	
	http.HandleFunc("/images/", func (w http.ResponseWriter, r *http.Request) {
		lsi := strings.LastIndex(r.URL.Path, "/")
		if lsi <= 8 {
			w.WriteHeader(400)
			return
		}
		info := getPieceInfo(r.URL.Path[8:lsi])
		if info == nil {
			w.WriteHeader(404)
			return
		}

		page, err := strconv.ParseUint(r.URL.Path[lsi+1:], 10, 16)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		blob := getPdfPage(info.PDF, uint(page))
		if blob == nil {
			w.WriteHeader(500)
			return
		}
		os.Stdout.Write(blob)
		w.Header().Set("Content-Type", "image/png")
		w.Write(blob)	
	})
	http.HandleFunc("/data/", func (w http.ResponseWriter, r *http.Request) {
		if strings.Index(r.URL.Path, "..") != -1 {
			w.WriteHeader(403)
			return
		}
		if !strings.HasSuffix(r.URL.Path, ".json") {
			w.WriteHeader(400)
			return
		}
		file, err := os.Open("." + r.URL.Path)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, file)
	})
	http.HandleFunc("/music/", func (w http.ResponseWriter, r *http.Request) {
		if strings.Index(r.URL.Path, ".") != -1 {
			w.WriteHeader(403)
			return
		}
		body, err := ioutil.ReadFile("music.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}

		info := getPieceInfo(r.URL.Path[7:])
		if info == nil {
			w.WriteHeader(404)
			return
		}
		mapraw, _ := json.Marshal(info.Map)
		body = []byte(strings.NewReplacer(
			"[@audio]", info.Audio,
			"[@name]", info.Name, 
			"[@map]", string(mapraw),
			).Replace(string(body)) )

		w.Header().Set("Content-Type", "text/html")		
		w.Write(body)
	})
	
	http.ListenAndServe(":8080", nil)
}