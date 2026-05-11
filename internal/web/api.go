package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"box-link/internal/app"
	"box-link/internal/logx"
)

const webLogLimit = 30
const eventLogBuffer = 32
const statusStreamInterval = 5 * time.Second

type APIResponse struct {
	OK      bool         `json:"ok"`
	Message string       `json:"message,omitempty"`
	Data    any          `json:"data,omitempty"`
	Logs    []logx.Entry `json:"logs,omitempty"`
}

type ConfigView struct {
	ConfigPath          string `json:"config_path"`
	Iface               string `json:"iface,omitempty"`
	HostIP              string `json:"host_ip"`
	BoxIP               string `json:"box_ip"`
	Netmask             string `json:"netmask"`
	HardwarePortPattern string `json:"hardware_port_pattern"`
	LogLevel            string `json:"log_level"`
	ListenAddr          string `json:"listen_addr"`
}

func registerAPI(mux *http.ServeMux, application *app.App) {
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{
			OK:      true,
			Message: "config loaded",
			Data: ConfigView{
				ConfigPath:          application.Config.ConfigPath,
				Iface:               application.Config.Iface,
				HostIP:              application.Config.HostIP,
				BoxIP:               application.Config.BoxIP,
				Netmask:             application.Config.Netmask,
				HardwarePortPattern: application.Config.HardwarePortPattern,
				LogLevel:            application.Config.LogLevel,
				ListenAddr:          application.Config.ListenAddr,
			},
			Logs: recentLogs(application),
		})
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		logCh, unsubscribe := application.LogStore.Subscribe(eventLogBuffer)
		defer unsubscribe()

		if err := writeSSE(w, flusher, "logs", recentLogs(application)); err != nil {
			return
		}
		if err := writeStatusEvent(r.Context(), w, flusher, application); err != nil {
			return
		}

		statusTicker := time.NewTicker(statusStreamInterval)
		defer statusTicker.Stop()

		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case entry, ok := <-logCh:
				if !ok {
					return
				}
				if err := writeSSE(w, flusher, "log", entry); err != nil {
					return
				}
			case <-statusTicker.C:
				if err := writeStatusEvent(r.Context(), w, flusher, application); err != nil {
					return
				}
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("/api/iface", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		result, err := application.ResolveIface(ctx)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: err.Error(), Logs: recentLogs(application)})
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{OK: true, Message: "interface resolved", Data: result, Logs: recentLogs(application)})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		result, err := application.Status(ctx)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: err.Error(), Logs: recentLogs(application)})
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{OK: true, Message: "status collected", Data: result, Logs: recentLogs(application)})
	})

	mux.HandleFunc("/api/doctor", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		result, err := application.Doctor(ctx)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: err.Error(), Logs: recentLogs(application)})
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{OK: true, Message: "diagnostics completed", Data: result, Logs: recentLogs(application)})
	})

	mux.HandleFunc("/api/on", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		result, err := application.On(ctx)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: err.Error(), Logs: recentLogs(application)})
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{OK: true, Message: "interface configured", Data: result, Logs: recentLogs(application)})
	})

	mux.HandleFunc("/api/off", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		result, err := application.Off(ctx)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: err.Error(), Logs: recentLogs(application)})
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{OK: true, Message: "interface restored to DHCP", Data: result, Logs: recentLogs(application)})
	})
}

func respondJSON(w http.ResponseWriter, status int, payload APIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func recentLogs(application *app.App) []logx.Entry {
	return application.LogStore.Recent(webLogLimit)
}

func writeStatusEvent(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, application *app.App) error {
	statusCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	result, err := application.Status(statusCtx)
	if err != nil {
		return writeSSE(w, flusher, "status", APIResponse{
			OK:      false,
			Message: err.Error(),
		})
	}

	return writeSSE(w, flusher, "status", APIResponse{
		OK:   true,
		Data: result,
	})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}

	flusher.Flush()
	return nil
}
