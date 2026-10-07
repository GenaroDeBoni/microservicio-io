# microservicio-io

Microservicio en Go para extracción de texto PDF, backend de extracción del monolito [`pdf-extractext`](https://github.com/AsterMaybe/pdf-extractext).

Recibe un PDF por `multipart/form-data`, extrae su texto y lo devuelve con metadata según el contrato esperado por `RemotePdfProcessor`. Los errores se responden en formato RFC 9457 (`application/problem+json`).

## Contrato HTTP

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Liveness/readiness → `200 {"status":"ok"}` |
| `POST` | `/api/v1/extract` | Extracción de texto; campo multipart `file` |

**Respuesta 200:**
```json
{"filename": "doc.pdf", "extension": ".pdf", "mime_type": "application/pdf", "text": "..."}
```

**Errores (RFC 9457):** 400 `invalid-file` · 422 `malformed-pdf` · 413 `too-large` · 504 `timeout` · 500 `server-error`

## Arquitectura

N-capas con dependencias hacia adentro y DIP en los límites:

```
main.go                → Composition root (rutas, env, wiring)
internal/handler/      → Presentación HTTP (extract, health)
internal/service/      → Lógica de negocio + puerto ExtractorPort
internal/apperr/       → Tipos de error + conversión RFC 9457
internal/model/        → DTOs de salida
external/pdf/          → Adaptador de infraestructura (ledongthuc/pdf)
```

`external/pdf` implementa `service.ExtractorPort`; el handler depende de su propia interfaz `Extractor`. Ninguna capa conoce la implementación concreta de la siguiente.

## Comandos

```bash
go test ./...              # Suite completa
go test -cover ./...       # Con cobertura (handler y service >= 80%)
go vet ./...               # Análisis estático
go build -o microservicio-io .   # Binario
docker compose build       # Imagen multi-stage
docker compose up -d       # Contenedor en la red compartida
```

## Configuración (variables de entorno)

| Variable | Default | Descripción |
|---|---|---|
| `PORT` | `8080` | Puerto HTTP |
| `PDF_MAX_SIZE_MB` | `15` | Límite de tamaño del PDF |
| `SHARED_NETWORK_NAME` | `default-shared-network` | Red Docker externa compartida con el monolito |

Ver `.env.example`.

## Testing

TDD (red → green): los tests se escriben antes que la implementación. Incluye tests unitarios por capa, table-driven tests de errores y tests de integración full-stack (`main_test.go`).

## Documentación

La especificación viva del proyecto está en [`SPEC-microservicio-io.md`](SPEC-microservicio-io.md) (objetivo, boundaries, success criteria y registro de decisiones).
