# microservicio-io

Microservicio en Go para **extracción de texto PDF**, backend de extracción del monolito [`pdf-extractext`](https://github.com/AsterMaybe/pdf-extractext).

Recibe un PDF por `multipart/form-data`, extrae su texto y lo devuelve con metadata según el contrato esperado por `RemotePdfProcessor`. Los errores se responden en formato [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) (`application/problem+json`).

---

## Índice

- [Especificaciones](#especificaciones)
- [Contrato HTTP](#contrato-http)
- [Uso del microservicio](#uso-del-microservicio)
- [Configuración](#configuración)
- [Arquitectura](#arquitectura)
- [Proyecto y comandos](#proyecto-y-comandos)
- [Testing](#testing)
- [Integración con el monolito](#integración-con-el-monolito)
- [Documentación](#documentación)

---

## Especificaciones

| Aspecto | Detalle |
|---|---|
| Lenguaje | Go 1.22 |
| Framework HTTP | [Gin](https://github.com/gin-gonic/gin) v1.10.0 |
| Extracción PDF | [ledongthuc/pdf](https://github.com/ledongthuc/pdf) (pure Go, sin CGo) |
| Errores | RFC 9457 — `application/problem+json` |
| Puerto | `8080` (configurable con `PORT`) |
| Límite de PDF | 15 MB por defecto (configurable con `PDF_MAX_SIZE_MB`) |
| Imagen Docker | Multi-stage: `golang:1.22-slim` → `alpine:3.20` |
| Cobertura de tests | handler ≥ 80% (86.2%), service ≥ 80% (100%), apperr 100% |

**Principios aplicados:** TDD (red → green), código limpio, SOLID (DIP en los límites), DRY, KISS, YAGNI y arquitectura en capas (n-layers).

---

## Contrato HTTP

### Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Liveness/readiness → `200 {"status":"ok"}` |
| `POST` | `/api/v1/extract` | Extracción de texto; campo multipart `file` |

### `POST /api/v1/extract` — Éxito (200)

Request: `multipart/form-data` con el campo **`file`** conteniendo los bytes del PDF.

```json
{
  "filename": "informe.pdf",
  "extension": ".pdf",
  "mime_type": "application/pdf",
  "text": "Texto extraído del documento…"
}
```

### Errores (RFC 9457)

Todos los errores responden `Content-Type: application/problem+json` con los campos `type`, `title`, `status`, `detail` e `instance`:

| HTTP | `type` | Cuándo ocurre |
|---|---|---|
| `400` | `/invalid-file` | Falta el campo `file`, está vacío o no comienza con los magic bytes `%PDF` |
| `413` | `/too-large` | El archivo supera `PDF_MAX_SIZE_MB` |
| `422` | `/malformed-pdf` | El PDF está corrupto o no se pudo extraer su texto |
| `504` | `/timeout` | Tiempo de extracción agotado |
| `500` | `/server-error` | Error interno no previsto |

Ejemplo de respuesta de error:

```json
{
  "type": "/invalid-file",
  "title": "invalid-file",
  "status": 400,
  "detail": "El archivo no es un PDF válido.",
  "instance": "/api/v1/extract"
}
```

---

## Uso del microservicio

### Requisitos

- Go ≥ 1.22 (para ejecutar en local)
- Docker + Docker Compose (para ejecutar en contenedor)

### Ejecutar en local

```bash
go run .                          # http://localhost:8080
PORT=9000 go run .                # puerto alternativo
PDF_MAX_SIZE_MB=25 go run .       # límite de tamaño alternativo
```

### Ejecutar con Docker

```bash
# Crear la red compartida la primera vez (si no existe):
docker network create default-shared-network

docker compose build
docker compose up -d

# Verificar:
curl http://localhost:8080/health
# → {"status":"ok"}
```

> `docker-compose.yml` usa la red externa `SHARED_NETWORK_NAME` (default:
> `default-shared-network`) para que el monolito alcance al microservicio como
> `http://microservicio-io:8080`. El servicio **no publica puertos** ni se expone
> vía Traefik: es un servicio interno (`traefik.enable=false`).

### Probar la extracción con curl

```bash
# Éxito:
curl -F "file=@documento.pdf" http://localhost:8080/api/v1/extract
# → {"filename":"documento.pdf","extension":".pdf","mime_type":"application/pdf","text":"…"}

# Archivo que no es PDF → 400:
curl -F "file=@notas.txt" http://localhost:8080/api/v1/extract
# → {"type":"/invalid-file",...,"status":400,...}

# Sin campo file → 400:
curl -X POST http://localhost:8080/api/v1/extract
```

Windows (PowerShell):

```powershell
curl.exe -F "file=@documento.pdf" http://localhost:8080/api/v1/extract
```

---

## Configuración

Variables de entorno (ver [`.env.example`](.env.example)):

| Variable | Default | Descripción |
|---|---|---|
| `PORT` | `8080` | Puerto HTTP del microservicio |
| `PDF_MAX_SIZE_MB` | `15` | Límite máximo de tamaño del PDF en MB |
| `SHARED_NETWORK_NAME` | `default-shared-network` | Red Docker externa compartida con el monolito |
| `TZ` | `UTC` | Zona horaria del contenedor |

---

## Arquitectura

N-capas con dependencias hacia adentro y DIP en los límites:

```
main.go                → Composition root (rutas, env, wiring)
internal/handler/      → Capa de presentación HTTP (extract, health)
internal/service/      → Capa de aplicación: lógica de negocio + puerto ExtractorPort
internal/apperr/       → Capa de dominio: tipos de error + conversión RFC 9457
internal/model/        → DTOs de salida (ExtractResponse, ProblemDetail)
external/pdf/          → Adaptador de infraestructura (ledongthuc/pdf)
```

- `external/pdf` **implementa** `service.ExtractorPort` (el servicio no conoce la librería concreta).
- `internal/handler` depende de su propia interfaz `Extractor` (el handler no conoce `*service.Service`).
- Validación de magic bytes `%PDF` tanto en el borde HTTP como en la regla de negocio (defensa en profundidad).

### Flujo de una petición

```
POST /api/v1/extract
  → handler: valida campo file + magic bytes
  → service: valida tamaño (413) y formato (400)
  → external/pdf: extrae texto con ledongthuc/pdf (422 si falla)
  → handler: responde 200 con metadata  |  RFC 9457 si hay error
```

---

## Proyecto y comandos

```
microservicio-io/
├── main.go                  # Composition root + rutas
├── main_test.go             # Tests de integración full-stack
├── internal/
│   ├── handler/             # Presentación HTTP (+ tests)
│   ├── service/             # Lógica de negocio + puerto (+ tests)
│   ├── apperr/              # Errores RFC 9457 (+ tests)
│   └── model/               # DTOs (+ tests)
├── external/
│   └── pdf/                 # Adaptador de extracción (+ tests)
├── docker/
│   └── Dockerfile           # Multi-stage build
├── docker-compose.yml       # Orquestación (red compartida, healthcheck)
├── .env.example             # Template de configuración
├── SPEC-microservicio-io.md # Especificación viva
└── README.md
```

```bash
go test ./...                    # Suite completa
go test -cover ./...             # Con cobertura (objetivo ≥ 80% en handler y service)
go vet ./...                     # Análisis estático
go build -o microservicio-io .   # Compilar binario
docker compose build             # Construir imagen
docker compose up -d             # Levantar contenedor
```

---

## Testing

Enfoque **TDD** (red → green): los tests se escriben antes que la implementación.

- **Unitarios por capa** — `internal/*/*_test.go` con table-driven tests para los 5 escenarios de error (400/413/422/504/500).
- **Integración full-stack** — `main_test.go`: PDF real → handler → service → adaptador → respuesta HTTP.
- **Hermético** — no requiere servicios externos ni Docker corriendo.
- **Smoke test manual** — verificado con `curl` contra el binario en vivo.

---

## Integración con el monolito

El monolito `pdf-extractext` consume este microservicio a través de
`RemotePdfProcessor` (`app/infrastructure/remote_pdf_processor.py`):

```python
# .env del monolito
PDF_EXTRACT_SERVICE_URL=http://microservicio-io:8080   # en Docker (red compartida)
DOCUMENT_GATEWAY_URL=http://microservicio-io:8080
EXTRACT_TIMEOUT_SECONDS=30
```

Flujo: el monolito valida magic bytes y checksum localmente, envía los bytes por
`POST {PDF_EXTRACT_SERVICE_URL}/api/v1/extract` (campo `file`) y recibe
`{"text": "…"}`. Los `400`/`422` se mapean a `InvalidPDFFormatError`; el resto a
`PDFProcessingError`.

---

## Documentación

- **Especificación viva**: [`SPEC-microservicio-io.md`](SPEC-microservicio-io.md) — objetivo, tech stack, boundaries (Always/Ask first/Never), success criteria y registro de decisiones.
- **Issues**: seguimiento del trabajo en la pestaña [Issues](../../issues) del repositorio.
