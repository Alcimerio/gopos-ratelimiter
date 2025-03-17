# Rate Limiter em Go

## Funcionalidades

- Limitação baseada em IP
- Limitação baseada em token (substitui os limites de IP)
- Armazenamento baseado em Redis com interface de armazenamento extensível
- Limites e durações de bloqueio configuráveis
- Middleware fácil de usar para servidores HTTP

## Configuração

A configuração é feita através de variáveis de ambiente ou arquivo `.env`:

```env
# Configuração do Limitador de Taxa
RATE_LIMIT_IP=5        # Máximo de requisições por segundo por IP
RATE_LIMIT_TOKEN=10    # Máximo de requisições por segundo por token
BLOCK_DURATION=300     # Duração do bloqueio em segundos (5 minutos)

# Configuração do Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

## Executando o Projeto

1. Inicie o Redis usando Docker Compose:
   ```bash
   docker-compose up -d
   ```

2. Instale as dependências:
   ```bash
   go mod download
   ```

3. Execute o servidor:
   ```bash
   go run main.go
   ```

O servidor será iniciado na porta 8080.

## Testando o Limitador de Taxa

Você pode testar o limitador de taxa usando curl:

1. Teste de limitação baseada em IP:
   ```bash
   curl http://localhost:8080/
   ```

2. Teste de limitação baseada em token:
   ```bash
   curl -H "API_KEY: seu-token" http://localhost:8080/
   ```

## Exemplos Detalhados de Uso

### Integração Básica com um Servidor HTTP

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/limiter"
	"github.com/alcimerio/gopos-ratelimiter/pkg/middleware"
	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
)

func main() {
	redisStorage, err := storage.NewRedisStorage("localhost", 6379, "", 0)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisStorage.Close()

	config := limiter.Config{
		IPLimit:       5,
		TokenLimit:    10,
		BlockDuration: 5 * time.Minute,
	}

	rateLimiter := limiter.NewRateLimiter(redisStorage, config)

	middleware := middleware.NewRateLimiterMiddleware(rateLimiter)

	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Olá, Mundo!"))
	})

	http.Handle("/", middleware.Handler(helloHandler))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Usando com o Router Gorilla Mux

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/limiter"
	"github.com/alcimerio/gopos-ratelimiter/pkg/middleware"
	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
	"github.com/gorilla/mux"
)

func main() {
	redisStorage, err := storage.NewRedisStorage("localhost", 6379, "", 0)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisStorage.Close()

	config := limiter.Config{
		IPLimit:       5,
		TokenLimit:    10,
		BlockDuration: 5 * time.Minute,
	}

	rateLimiter := limiter.NewRateLimiter(redisStorage, config)
	rateLimiterMiddleware := middleware.NewRateLimiterMiddleware(rateLimiter)

	r := mux.NewRouter()

	publicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ponto de extremidade público - Limitação de taxa baseada em IP"))
	})
	r.Handle("/public", rateLimiterMiddleware.Handler(publicHandler))

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ponto de extremidade da API - Limitação de taxa baseada em token"))
	})
	r.Handle("/api", rateLimiterMiddleware.Handler(apiHandler))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
```

### Usando com uma Storage personalizado

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/limiter"
	"github.com/alcimerio/gopos-ratelimiter/pkg/middleware"
	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
)

type CustomStorage struct {
	// Your storage fields here
}

func NewCustomStorage() *CustomStorage {
	return &CustomStorage{}
}

// Implement the storage.Storage interface
func (c *CustomStorage) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	return 1, nil
}

func (c *CustomStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (c *CustomStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	return nil
}

func (c *CustomStorage) Reset(ctx context.Context, key string) error {
	return nil
}

func (c *CustomStorage) Close() error {
	return nil
}

func main() {
	customStorage := NewCustomStorage()
	config := limiter.Config{
		IPLimit:       5,
		TokenLimit:    10,
		BlockDuration: 5 * time.Minute,
	}

	rateLimiter := limiter.NewRateLimiter(customStorage, config)
	middleware := middleware.NewRateLimiterMiddleware(rateLimiter)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Olá, Mundo!"))
	})

	http.Handle("/", middleware.Handler(handler))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Executando Testes

Para executar todos os testes:

```bash
go test ./...
```

Para executar testes com cobertura:

```bash
go test -cover ./...
```

Para gerar um relatório detalhado de cobertura:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Nota: Os testes do Redis requerem uma instância do Redis em execução. Certifique-se de que o Redis esteja rodando antes de executar os testes.

## Arquitetura

O projeto segue uma abordagem de arquitetura limpa com os seguintes componentes:

- `pkg/storage`: Interface de armazenamento e implementação do Redis
- `pkg/limiter`: Lógica principal de limitação de taxa
- `pkg/middleware`: Middleware HTTP para limitação de taxa

A interface de armazenamento permite fácil extensão para suportar outros backends de armazenamento além do Redis.
