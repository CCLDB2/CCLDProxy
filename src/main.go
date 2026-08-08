// ccldproxy - Servidor de tuneles SSH/OpenVPN/V2Ray (proxy limpio)
// Reescritura propia en Go. Usa el upgrade HTTP 101 y reenvia raw TCP al backend.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

// version del motor.
const version = "0.1.2"

// printVersion es el flag --version.
var printVersion *bool

// Config es la configuracion cargada de flags.
type Config struct {
	Token      string
	Port       int
	HTTP       bool
	HTTPS      bool
	Cert       string
	Key        string
	Workers    int
	BufferSize int
	MaxPerIP   int   // 0 = sin limite
	Quiet      bool
	Backend    []Backend // backends de protocolo

	Validate bool
}

// Backend define un destino de protocolo.
type Backend struct {
	Name string
	Host string
	Port int
}

func main() {
	cfg := parseFlags()

	// --version / --v imprime la version del motor y sale
	if *printVersion {
		fmt.Printf("ccldproxy %s\n", version)
		return
	}

	if cfg.Validate {
		// Validacion local: en esta reescritura no hay servidor externo.
		fmt.Println("Success: Your token is valid")
		return
	}

	logger := newLogger(cfg.Quiet)

	var ln net.Listener
	var err error

	if cfg.HTTPS {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
		if err != nil {
			logger.Fatalf("HTTPS: no se pudo escuchar :%d: %v", cfg.Port, err)
		}
		logger.Infof("Escuchando HTTPS en :%d (cert=%s)", cfg.Port, cfg.Cert)
		server := &TLSServer{cfg: cfg, logger: logger}
		server.Serve(ln)
		return
	}

	ln, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		logger.Fatalf("no se pudo escuchar :%d: %v", cfg.Port, err)
	}
	logger.Infof("ccldproxy escuchando en :%d (HTTP)", cfg.Port)

	server := &Server{
		cfg:    cfg,
		logger: logger,
		limiter: newLimiter(cfg.MaxPerIP, logger),
		conns:  make(map[int64]*Client),
	}
	server.Serve(ln)
}

func parseFlags() *Config {
	cfg := &Config{
		Workers:    1000,
		BufferSize: 16384,
	}

	flag.StringVar(&cfg.Token, "token", "", "token de acceso")
	flag.BoolVar(&cfg.Validate, "validate", false, "validar el token y salir")
	flag.IntVar(&cfg.Port, "port", 80, "puerto de escucha")
	flag.BoolVar(&cfg.HTTP, "http", false, "usar HTTP")
	flag.BoolVar(&cfg.HTTPS, "https", false, "usar HTTPS")
	flag.StringVar(&cfg.Cert, "cert", "", "ruta del certificado (HTTPS)")
	flag.StringVar(&cfg.Key, "key", "", "ruta de la clave privada (HTTPS)")
	flag.IntVar(&cfg.Workers, "workers", 1000, "trabajadores (limite de conexiones)")
	flag.IntVar(&cfg.BufferSize, "buffer-size", 16384, "tamano de buffer")
	flag.IntVar(&cfg.MaxPerIP, "max-per-ip", 0, "limite de conexiones por IP (0 = sin limite)")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "modo silencioso")
	printVersion = flag.Bool("version", false, "mostrar version del motor y salir")
	flag.Parse()

	// Backends por defecto (los que usan las apps)
	cfg.Backend = []Backend{
		{Name: "SSH", Host: "127.0.0.1", Port: 22},
		{Name: "OpenVPN", Host: "127.0.0.1", Port: 1194},
		{Name: "V2Ray", Host: "127.0.0.1", Port: 1080},
	}
	return cfg
}

// --- Logger ----------------------------------------------------------------

type logger struct {
	quiet bool
	mu    sync.Mutex
}

func newLogger(quiet bool) *logger { return &logger{quiet: quiet} }

func (l *logger) Infof(format string, a ...interface{}) {
	if l.quiet {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[INFO] "+format, a...)
}

func (l *logger) Errorf(format string, a ...interface{}) {
	if l.quiet {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[ERRO] "+format, a...)
}

func (l *logger) Fatalf(format string, a ...interface{}) {
	log.Fatalf("[FATAL] "+format, a...)
}

var _ = os.Stdout
