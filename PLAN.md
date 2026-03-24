# PLAN

## 1. Propósito del Documento

Este archivo es la hoja de ruta técnica para migrar `agent47` desde su runtime actual Bash-first a un runtime principal en Go, preservando:

- el contrato funcional actual del producto
- la compatibilidad fuerte con macOS
- el flujo de uso actual basado en `afs`
- el modelo de scaffold gestionado por templates + manifest

Y agregando:

- soporte real para Windows
- distribución portable por binarios
- testing cross-platform
- una base de código más mantenible y menos dependiente del shell

Este documento no es solo un plan conceptual. Debe servir como guía de implementación.

Estado base del plan:

- release actual del proyecto: `v1.0.22`
- nombre del producto: `agent47`
- comando público actual: `afs`

## 2. Resumen Ejecutivo

La migración propuesta no es un “rewrite big bang” del proyecto completo. Es una migración controlada por paridad funcional, donde:

1. Se congela el comportamiento observable del producto actual.
2. Se construye un nuevo runtime en Go con límites de arquitectura claros.
3. Se mantienen `templates/`, `manifest.txt`, `README.md`, `docs/`, `SPEC.md` y `SNAPSHOT.md` como contratos documentales del producto.
4. Se preserva la experiencia principal de instalación en macOS/Linux mediante `install.sh`, pero `install.sh` deja de contener la lógica core del producto.
5. En Windows se habilita una instalación nativa y funcional sin Bash, sin Git Bash y sin WSL.
6. El producto oficial pasa a ser un binario Go, pero el desarrollo desde GitHub sigue siendo cómodo porque el repo conserva los templates en filesystem y soporta un modo explícito de desarrollo.

## 3. Objetivos de Migración

### 3.1 Objetivos funcionales

- Mantener todos los comandos públicos actuales:
  - `afs help`
  - `afs uninstall`
  - `afs doctor [--check-update|--check-update-force]`
  - `afs add-agent [--force] [--only-skills]`
  - `afs add-agent-prompt [--force]`
  - `afs add-ss-prompt`
- Mantener el mismo modelo de scaffold:
  - `AGENTS.md`
  - `rules/*.yaml`
  - `skills/*`
  - `skills/AVAILABLE_SKILLS.xml`
  - prompts
  - `specs/spec.yml` como artifact template, no scaffold por defecto
- Mantener la semántica de ownership:
  - managed targets
  - preserved targets
  - refresh con `--force`
  - rollback ante fallos

### 3.2 Objetivos técnicos

- Sustituir dependencias del shell por primitivas Go.
- Diseñar el runtime para macOS/Linux/Windows desde la base.
- Tener un pipeline de tests y releases multiplataforma.
- Evitar drift entre binario y templates en releases.

### 3.3 Objetivos de UX

- Mantener `afs` como único comando público.
- Mantener el output casi idéntico al actual.
- Mantener la experiencia actual de macOS lo más intacta posible.
- Hacer que Windows sea usable sin conocimientos Unix.

## 4. No Objetivos

- Rehacer el producto como framework general de project scaffolding.
- Cambiar el contrato documental del scaffold salvo necesidad explícita.
- Añadir backend remoto o servicios cloud.
- Cambiar el formato de `templates/manifest.txt` durante la fase inicial.
- Introducir `.pkg`, `.exe` installer o `.msi` en la primera etapa.

## 5. Decisiones Cerradas

Estas decisiones ya no se discuten en esta migración.

### 5.1 Comando público

- `afs` será el único comando público.

### 5.2 Runtime oficial

- El runtime oficial será un binario Go.

### 5.3 Modelo de templates

- Releases oficiales:
  - templates embebidos en el binario Go.
- Desarrollo local:
  - modo explícito para leer `templates/` desde filesystem del repo.

### 5.4 Instalación

- macOS/Linux:
  - `install.sh` sigue siendo la experiencia principal.
  - `install.sh` pasa a instalar el binario Go y el runtime, no a ejecutar la lógica del producto.
  - `install.sh --non-interactive` se preserva como contrato público para automatización y entornos no interactivos.
- Windows:
  - instalación inicial simple, sin `.exe` installer ni `.msi`.

### 5.5 Compatibilidad macOS

- macOS debe preservar experiencia y comportamiento sustancialmente equivalentes al estado actual.

### 5.6 Output

- El output del CLI debe ser casi idéntico al actual.
- Se permiten diferencias menores cuando sean necesarias por plataforma o claridad.

### 5.7 Formato de manifest

- `templates/manifest.txt` se conserva en la migración inicial.

### 5.8 Helpers publicados

- La migración a Go debe seguir publicando helper scripts instalables:
  - `add-agent`
  - `add-agent-prompt`
  - `add-ss-prompt`

### 5.9 Compatibilidad legacy

- `init-agent` no se considera necesario como compat path a preservar en Go.
- Su eliminación ya se acepta como cambio deliberado de producto durante la migración.

## 6. Estado Actual del Producto

El producto actual, según `README.md`, `SNAPSHOT.md`, `SPEC.md`, `docs/` y el comportamiento ya validado del repo, tiene estas propiedades:

- CLI Bash-first
- runtime instalado en `~/.agent47`
- entrada pública vía `afs`
- scaffold basado en templates
- ownership explícito vía manifest
- suite Bats como validación principal
- comportamiento transaccional en bootstrap e instalación

El proyecto hoy separa razonablemente:

- router (`bin/afs`)
- runtime env
- installer
- bootstrap engine
- doctor/update
- skill validation
- docs
- templates

La migración debe preservar esa separación conceptual, aunque cambie el lenguaje.

## 7. Problemas Técnicos del Estado Actual

### 7.1 Dependencia fuerte de shell

La implementación actual depende de:

- Bash
- `cp`
- `mv`
- `find`
- `grep`
- `sed`
- `awk`
- `mktemp`
- `readlink`
- shell rc files

### 7.2 Portabilidad limitada

Esto dificulta:

- soporte real de Windows
- distribución como producto
- testing multiplataforma homogéneo
- packaging robusto

### 7.3 Riesgo de drift entre runtime y assets

El runtime actual depende de archivos instalados en disco, lo que complica:

- instalaciones incompletas
- assets faltantes
- diferencias entre repo local y runtime instalado

## 8. Resultado Esperado al Final de la Migración

Al final del proceso, el proyecto debería verse conceptualmente así:

```text
user
  |
  +--> ./install.sh              # macOS/Linux
  |      installs Go binary
  |
  +--> simple install flow       # Windows
  |      installs afs.exe
  |
  `--> afs
         |
         +--> doctor
         +--> add-agent
         +--> add-agent --force
         +--> add-agent --only-skills
         +--> add-agent-prompt
         `--> add-ss-prompt
```

Y a nivel de implementación:

- CLI en Go
- templates embebidos en releases
- modo dev leyendo templates del repo
- tests Go + smoke tests
- CI matrix en macOS/Linux/Windows

## 9. Arquitectura Objetivo

## 9.1 Layout del repo

```text
agent47/
|
+-- cmd/
|   `-- afs/
|       `-- main.go
|
+-- internal/
|   +-- app/
|   |   +-- root.go
|   |   +-- install.go
|   |   +-- uninstall.go
|   |   +-- doctor.go
|   |   +-- add_agent.go
|   |   +-- add_agent_prompt.go
|   |   `-- add_ss_prompt.go
|   +-- cli/
|   |   +-- parser.go
|   |   `-- output.go
|   +-- runtime/
|   |   +-- config.go
|   |   +-- dirs.go
|   |   `-- mode.go
|   +-- templates/
|   |   +-- source.go
|   |   +-- embedded.go
|   |   +-- filesystem.go
|   |   `-- loader.go
|   +-- manifest/
|   |   +-- manifest.go
|   |   +-- parser.go
|   |   +-- validate.go
|   |   `-- sections.go
|   +-- fsx/
|   |   +-- files.go
|   |   +-- dirs.go
|   |   +-- atomic.go
|   |   +-- backup.go
|   |   `-- transaction.go
|   +-- skills/
|   |   +-- frontmatter.go
|   |   +-- validate.go
|   |   +-- discover.go
|   |   `-- xml.go
|   +-- doctor/
|   |   +-- doctor.go
|   |   +-- install_check.go
|   |   +-- template_check.go
|   |   `-- path_check.go
|   +-- update/
|   |   +-- cache.go
|   |   +-- release_check.go
|   |   +-- git_check.go
|   |   `-- service.go
|   +-- platform/
|   |   +-- common.go
|   |   +-- darwin.go
|   |   +-- linux.go
|   |   +-- windows.go
|   |   +-- clipboard.go
|   |   `-- path_hint.go
|   +-- version/
|   |   `-- version.go
|   `-- testutil/
|
+-- templates/
+-- docs/
+-- scripts/
+-- testdata/
+-- go.mod
+-- go.sum
`-- Makefile
```

## 9.2 Responsabilidades por paquete

### `cmd/afs`

Responsabilidades:

- `main`
- wiring del runtime
- traducción de argumentos a opciones tipadas
- código de salida final

No debe contener:

- lógica de negocio
- filesystem logic compleja
- manifest parsing

### `internal/app`

Responsabilidades:

- casos de uso del producto
- coordinación entre subsistemas
- ejecución de comandos

Debe ser la capa que responde preguntas como:

- qué hace `add-agent --force`
- cuándo abortar
- cuándo hacer rollback

### `internal/runtime`

Responsabilidades:

- resolver directorios de runtime
- distinguir modo release vs modo desarrollo
- exponer configuración compartida

### `internal/templates`

Responsabilidades:

- abstraer la fuente de templates
- cargar assets embebidos
- cargar assets desde filesystem
- entregar contenido con una interfaz uniforme

### `internal/manifest`

Responsabilidades:

- parsear `manifest.txt`
- validar contrato
- exponer secciones tipadas

### `internal/fsx`

Responsabilidades:

- operaciones de filesystem cross-platform
- abstracciones de atomic write y staged replacement
- backups y rollback

### `internal/skills`

Responsabilidades:

- parsear frontmatter
- validar shape de skills
- descubrir skills
- generar `AVAILABLE_SKILLS.xml`

### `internal/doctor`

Responsabilidades:

- diagnósticos locales
- checks del runtime instalado
- checks de templates
- checks de integridad

### `internal/update`

Responsabilidades:

- update cache
- check remoto por release/version
- check git para modo desarrollo

### `internal/platform`

Responsabilidades:

- diferencias entre macOS/Linux/Windows
- resolución de dirs
- clipboard
- hints de PATH
- comportamiento opcional OS-specific

## 10. Contratos que Deben Preservarse

## 10.1 Comandos públicos

Se deben mantener:

```text
afs help
afs uninstall
afs doctor
afs doctor --check-update
afs doctor --check-update-force
afs add-agent
afs add-agent --force
afs add-agent --only-skills
afs add-agent --only-skills --force
afs add-agent-prompt [--force]
afs add-ss-prompt
```

## 10.2 Semántica de scaffold

`afs add-agent` debe:

- escribir `AGENTS.md` si no existe
- escribir `rules/*.yaml` si no existen
- crear `skills/*`
- generar `skills/AVAILABLE_SKILLS.xml`
- crear `README.md` vacío si no existe
- no scaffoldear `specs/spec.yml` al target por defecto

`afs add-agent --force` debe:

- reconciliar managed targets contra templates actuales
- eliminar stale managed rules/skills
- preservar:
  - `README.md`
  - `specs/spec.yml`
  - `SNAPSHOT.md`
  - `SPEC.md`

## 10.3 Ownership model

Managed targets:

- `AGENTS.md`
- `rules/*.yaml`
- `skills/*`
- `skills/AVAILABLE_SKILLS.xml`

Preserved targets:

- `README.md`
- `specs/spec.yml`
- `SNAPSHOT.md`
- `SPEC.md`

## 10.4 Update behavior

- `doctor` no chequea updates por defecto
- `doctor --check-update` sí
- `doctor --check-update-force` ignora el cache y rehace la comprobación
- el cache debe tener TTL
- un cache corrupto debe ignorarse sin romper el comando
- el cache debe invalidarse si cambia la versión local
- si el repo es un checkout git:
  - el check usa git como fuente primaria
  - los hints apuntan a `git pull` + `./install.sh`
- si no hay checkout git o el fetch falla:
  - se intenta la lectura remota de `VERSION`
  - los hints apuntan a reinstalar desde una release publicada
- el fallo de herramientas opcionales o de red no debe romper el CLI: degrada a warning

## 10.5 Contrato de `doctor`

El runtime Go debe preservar también estas capacidades del diagnóstico actual:

- verificar que `afs` en `PATH` sea el launcher gestionado esperado
- verificar que los helpers públicos instalados estén disponibles:
  - `add-agent`
  - `add-agent-prompt`
  - `add-ss-prompt`
- advertir sobre scripts legacy cuando aparezcan en ubicaciones gestionadas o visibles en `PATH`
- verificar layout de templates y prompts instalados
- seguir degradando a warnings cuando faltan herramientas opcionales o el entorno no permite validaciones completas

El matching de `doctor` debe preservar semántica estricta de paridad:

- matching por resolución real de path/symlink para `afs`
- matching por resolución o equivalencia de contenido para helpers publicados cuando corresponda
- hints y warnings compatibles con la UX actual

Matriz mínima de matching:

| Target | Estado OK | Estado WARN |
| --- | --- | --- |
| `afs` en `PATH` | `command -v afs` resuelve al mismo destino real que `~/.agent47/bin/afs` | existe en `PATH` pero resuelve a otro destino |
| `add-agent` / `add-agent-prompt` / `add-ss-prompt` | el comando resuelve al helper gestionado o a la copia visible en `~/bin` con contenido idéntico a la copia gestionada | existe en `PATH` pero no coincide ni por destino real ni por contenido |
| `~/bin/afs` | symlink que resuelve al launcher gestionado | symlink roto, apunta a otro binario, o falta |
| templates | `~/.agent47/templates` existe y pasa checks de manifest/prompts/security/AGENTS | estructura faltante o inválida |
| legacy helpers | ausentes | presentes en `PATH` o ubicaciones gestionadas, con hint de cleanup |

## 10.6 Contrato de instalación

La instalación Go debe preservar estos comportamientos observables:

- `install.sh --non-interactive` sigue existiendo y evita modificaciones interactivas del shell rc
- si no hay TTY, el instalador entra en modo no interactivo aunque no se pase el flag
- instalación sin `--force` no pisa launchers, symlinks o helpers existentes cuando hoy se preservan
- instalación con `--force` refresca runtime gestionado y limpia legacy scripts según el contrato actual
- uninstall limpia launcher, helpers publicados y scripts legacy gestionados

Matriz mínima de install/uninstall:

| Target | Install sin `--force` | Install con `--force` | Uninstall |
| --- | --- | --- | --- |
| `~/.agent47/bin/afs` | si existe, warn y preservar | reemplazar con launcher gestionado | eliminar |
| `~/.agent47/scripts/<helper>` | si existe, warn y preservar | reemplazar helper gestionado | eliminar |
| `~/.agent47/scripts/lib` | si existe, warn y preservar | refrescar librería gestionada | eliminar |
| `~/.agent47/templates` | si existe, warn y preservar | swap atómico con backup | eliminar |
| `~/bin/<helper>` | si existe, warn y preservar | publicar copia visible con rollback por helper | eliminar |
| `~/bin/afs` | si existe cualquier entrada, warn y preservar | refrescar symlink al launcher gestionado con rollback del link previo | eliminar symlink |
| legacy scripts (`a47`, prompt helpers legacy) | limpiar solo ubicaciones gestionadas/visibles del runtime | limpiar | eliminar si existen |

Invariants de bootstrap/rollback que el port a Go debe preservar:

- `README.md`:
  - se crea vacío solo si falta
  - en rollback se elimina solo si fue creado por la operación
- `AGENTS.md`:
  - sin `--force`, se preserva si ya existe
  - con `--force`, se reemplaza y se restaura desde backup si falla la transacción
- `rules/*.yaml`:
  - sin `--force`, no pisa archivos existentes
  - con `--force`, actualiza managed rules y elimina stale managed rules
  - rollback por archivo
- `skills/`:
  - sin `--force`, parte del árbol existente y preserva contenido local cuando corresponde
  - con `--force`, reemplaza el árbol gestionado completo
  - rollback por directorio completo
- preserved targets:
  - `README.md`
  - `specs/spec.yml`
  - `SNAPSHOT.md`
  - `SPEC.md`
  no se tocan durante refresh

## 10.7 Output style

Mantener estilo similar a:

```text
[INFO] ...
[WARN] ...
[ERR] ...
[OK] ...
```

## 11. Layout Operativo por Plataforma

## 11.1 macOS

```text
~/.agent47/
  bin/
    afs
  cache/
    update.cache
  ...

~/bin/
  afs
```

Decisión:

- conservar experiencia actual
- `install.sh` sigue siendo el entrypoint principal del usuario

## 11.2 Linux

```text
~/.agent47/
  bin/
    afs
  cache/
    update.cache
  ...

~/bin/
  afs
```

Decisión:

- misma estrategia que macOS

## 11.3 Windows

```text
%LOCALAPPDATA%\agent47\
  bin\
    afs.exe
  cache\
    update.cache
  ...
```

Decisión:

- no usar `~/bin`
- no usar symlink como mecanismo central
- agregar `%LOCALAPPDATA%\agent47\bin` al PATH del usuario

## 12. Modelo de Templates

## 12.1 Requisito

El producto necesita usar los mismos templates en:

- releases
- tests
- desarrollo local

## 12.2 Estrategia elegida

Modelo híbrido.

### Releases

- templates embebidos con `embed`

### Desarrollo local

- filesystem templates desde `./templates`

### Selección de fuente

Debe ser explícita.

Ejemplo:

```go
type TemplateMode int

const (
    TemplateModeEmbedded TemplateMode = iota
    TemplateModeFilesystem
)

type Source interface {
    ReadFile(path string) ([]byte, error)
    ReadDir(path string) ([]DirEntry, error)
    Stat(path string) (FileInfo, error)
}
```

## 12.3 Recomendación de implementación

```go
type Loader struct {
    Source Source
}

func NewLoader(mode TemplateMode, repoRoot string) (*Loader, error) {
    switch mode {
    case TemplateModeEmbedded:
        return &Loader{Source: NewEmbeddedSource()}, nil
    case TemplateModeFilesystem:
        return &Loader{Source: NewFilesystemSource(filepath.Join(repoRoot, "templates"))}, nil
    default:
        return nil, fmt.Errorf("unknown template mode")
    }
}
```

## 13. Diseño del Runtime

## 13.1 Config principal

```go
type Config struct {
    OS               string
    HomeDir          string
    UserBinDir       string
    Agent47Home      string
    CacheDir         string
    UpdateCacheFile  string
    Version          string
    TemplateMode     TemplateMode
    RepoRoot         string
}
```

## 13.2 Constructor recomendado

```go
func DetectConfig(env Env, pf platform.Service) (Config, error) {
    home, err := pf.HomeDir()
    if err != nil {
        return Config{}, err
    }

    var agentHome string
    var userBin string

    if pf.IsWindows() {
        localAppData, err := pf.LocalAppDataDir()
        if err != nil {
            return Config{}, err
        }
        agentHome = filepath.Join(localAppData, "agent47")
        userBin = filepath.Join(agentHome, "bin")
    } else {
        agentHome = filepath.Join(home, ".agent47")
        userBin = filepath.Join(home, "bin")
    }

    return Config{
        OS:              pf.OS(),
        HomeDir:         home,
        UserBinDir:      userBin,
        Agent47Home:     agentHome,
        CacheDir:        filepath.Join(agentHome, "cache"),
        UpdateCacheFile: filepath.Join(agentHome, "cache", "update.cache"),
        Version:         version.Current(),
    }, nil
}
```

## 14. CLI Design

## 14.1 Estrategia

Preferir stdlib `flag` o parser propio mínimo antes que introducir un framework pesado.

Razones:

- superficie pequeña
- comandos limitados
- control fino del output
- menos dependencias

## 14.2 Comandos

Ejemplo de estructura:

```go
type Command interface {
    Name() string
    Run(ctx context.Context, args []string) error
}
```

Router:

```go
func Run(ctx context.Context, cfg Config, args []string) int {
    if len(args) == 0 || args[0] == "help" {
        printHelp(cfg.Version)
        return 0
    }

    switch args[0] {
    case "doctor":
        return runDoctor(ctx, cfg, args[1:])
    case "uninstall":
        return runUninstall(ctx, cfg, args[1:])
    case "add-agent":
        return runAddAgent(ctx, cfg, args[1:])
    case "add-agent-prompt":
        return runAddAgentPrompt(ctx, cfg, args[1:])
    case "add-ss-prompt":
        return runAddSSPrompt(ctx, cfg, args[1:])
    default:
        fmt.Printf("Unknown command: %s\n", args[0])
        printHelp(cfg.Version)
        return 1
    }
}
```

## 14.3 Output

Crear un wrapper pequeño:

```go
type Output struct {
    Stdout io.Writer
    Stderr io.Writer
}

func (o Output) Info(msg string, args ...any) { fmt.Fprintf(o.Stdout, "[INFO] "+msg+"\n", args...) }
func (o Output) Warn(msg string, args ...any) { fmt.Fprintf(o.Stdout, "[WARN] "+msg+"\n", args...) }
func (o Output) OK(msg string, args ...any)   { fmt.Fprintf(o.Stdout, "[OK] "+msg+"\n", args...) }
func (o Output) Err(msg string, args ...any)  { fmt.Fprintf(o.Stderr, "[ERR] "+msg+"\n", args...) }
```

## 15. Diseño de Install / Uninstall

## 15.1 Requisito funcional

Instalar:

- runtime dir
- binario gestionado
- cache dir
- launcher visible al usuario
- helper scripts publicados

Desinstalar:

- binario visible
- binario gestionado
- cache
- runtime data gestionada
- helper scripts publicados

## 15.2 Reglas por plataforma

### macOS/Linux

- copiar binario a `~/.agent47/bin/afs`
- publicar helpers en `~/.agent47/scripts/`
- crear `~/bin/afs` como symlink al binario gestionado
- publicar helpers visibles en `~/bin/` con la misma semántica actual
- si `~/bin` no está en `PATH`, dar hint o actualizar shell rc desde `install.sh`

Decisión cerrada:

- en Unix se preserva el modelo actual basado en symlink para mantener paridad con `doctor`, uninstall y UX actual
- no se usará wrapper script salvo que aparezca una limitación concreta por plataforma

### Windows

- copiar `afs.exe` a `%LOCALAPPDATA%\agent47\bin\afs.exe`
- publicar helpers equivalentes si la migración mantiene ese contrato también en Windows
- agregar `%LOCALAPPDATA%\agent47\bin` al `PATH` del usuario

## 15.3 Diseño de API

```go
type Installer struct {
    FS       fsx.Service
    Platform platform.Service
    Out      Output
}

type InstallOptions struct {
    Force bool
}

func (i *Installer) Install(ctx context.Context, cfg Config, opts InstallOptions) error
func (i *Installer) Uninstall(ctx context.Context, cfg Config) error
```

## 15.4 Pseudocódigo de instalación

```go
func (i *Installer) Install(ctx context.Context, cfg Config, opts InstallOptions) error {
    if err := i.FS.MkdirAll(filepath.Join(cfg.Agent47Home, "bin")); err != nil {
        return err
    }
    if err := i.FS.MkdirAll(cfg.CacheDir); err != nil {
        return err
    }

    if err := i.installManagedBinary(cfg, opts); err != nil {
        return err
    }

    if cfg.OS == "windows" {
        if err := i.Platform.EnsureUserPath(cfg.UserBinDir); err != nil {
            return err
        }
    } else {
        if err := i.installUnixLauncher(cfg, opts); err != nil {
            return err
        }
    }

    i.Out.OK("afs installation complete")
    return nil
}
```

## 15.5 Riesgos técnicos

- `rename` no siempre equivale a Unix en Windows
- symlink en Windows puede requerir permisos especiales
- modificación de `PATH` en Windows debe ser estable y reversible

## 16. Diseño de Bootstrap

## 16.1 Requisito funcional

`add-agent` debe:

- validar templates requeridos
- validar soporte de skills
- crear staging
- escribir files managed según reglas actuales
- generar `AVAILABLE_SKILLS.xml`
- hacer rollback si algo falla

## 16.2 API sugerida

```go
type BootstrapService struct {
    FS        fsx.Service
    Loader    *templates.Loader
    Manifest  manifest.Manifest
    Skills    skills.Service
    Out       Output
}

type BootstrapOptions struct {
    Force      bool
    OnlySkills bool
    WorkDir    string
}

func (b *BootstrapService) Run(ctx context.Context, opts BootstrapOptions) error
```

## 16.3 Modelo transaccional

Necesitamos abstraer:

- `stageDir`
- `backupDir`
- `commit`
- `rollback`

Interfaz recomendada:

```go
type Transaction interface {
    StageFile(rel string, data []byte) error
    BackupPath(path string) error
    ReplaceFile(target string, staged string) error
    ReplaceDir(target string, staged string) error
    Commit() error
    Rollback() error
}
```

## 16.4 Algoritmo de `add-agent`

```text
1. Resolver templates loader
2. Parsear manifest
3. Validar contract
4. Validar templates requeridos
5. Validar skills support
6. Crear transaction
7. Stage de skills
8. Si !only-skills:
   - stage rules
   - stage AGENTS.md
9. Commit:
   - rules
   - AGENTS
   - README vacío si falta
   - skills
10. Commit transaction
11. Ante error:
   - rollback
```

## 16.5 Comportamientos delicados

### README

Mantener:

- si falta, se crea vacío
- no se copia desde template

### `specs/spec.yml`

Mantener:

- existe como template asset
- no se scaffoldéa automáticamente

### `--force`

Mantener:

- reconciliación real
- remoción de stale managed files bajo paths gestionados
- preservación de preserved targets

## 16.6 Pseudocódigo parcial

```go
func (b *BootstrapService) Run(ctx context.Context, opts BootstrapOptions) (err error) {
    tx, err := b.FS.NewTransaction(opts.WorkDir)
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    if err = b.stageSkills(tx, opts); err != nil {
        return err
    }

    if !opts.OnlySkills {
        if err = b.stageRules(tx, opts); err != nil {
            return err
        }
        if err = b.stageAgents(tx, opts); err != nil {
            return err
        }
    }

    if err = b.commit(tx, opts); err != nil {
        return err
    }

    return tx.Commit()
}
```

## 17. Diseño de Skills

## 17.1 Requisitos

Validar que `SKILL.md`:

- exista
- tenga frontmatter
- incluya `name`
- incluya `description`
- `name` sea kebab-case
- `name` no exceda longitud
- `description` no exceda longitud

## 17.2 API sugerida

```go
type Skill struct {
    Name        string
    Description string
    Location    string
}

type Service struct{}

func (Service) Validate(path string, body []byte) error
func (Service) Discover(src templates.Source, root string) ([]Skill, error)
func (Service) GenerateAvailableSkillsXML(skills []Skill) ([]byte, error)
```

## 17.3 Ejemplo de parsing

```go
type Frontmatter struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
}
```

## 17.4 XML generation

```go
type availableSkills struct {
    XMLName xml.Name    `xml:"available_skills"`
    Skills  []xmlSkill  `xml:"skill"`
}

type xmlSkill struct {
    Name        string `xml:"name"`
    Description string `xml:"description"`
    Location    string `xml:"location"`
}
```

## 18. Diseño de Manifest

## 18.1 Problema

`manifest.txt` no es YAML/JSON. Debe parsearse como secciones tipo INI muy simples.

## 18.2 Modelo tipado

```go
type Manifest struct {
    RuleTemplates        []string
    ManagedTargets       []string
    PreservedTargets     []string
    RequiredTemplateFiles []string
    RequiredTemplateDirs  []string
}
```

## 18.3 Parser

```go
func Parse(data []byte) (Manifest, error)
func (m Manifest) Validate() error
func (m Manifest) ContainsRuleTemplate(name string) bool
```

## 18.4 Validaciones mínimas

- secciones requeridas presentes
- secciones requeridas no vacías
- entries normalizadas

## 19. Diseño de Doctor

## 19.1 Responsabilidades

- verificar que `afs` apunta al runtime esperado
- verificar que el launcher Unix o el path visible del usuario apunta al target correcto
- verificar runtime assets mínimos
- verificar presencia de helpers públicos instalados
- detectar scripts legacy relevantes y reportarlos como warning
- verificar templates y manifest
- verificar security rule IDs duplicados
- verificar secciones requeridas de `AGENTS.md`
- verificar update status opcional

## 19.2 API sugerida

```go
type Doctor struct {
    FS       fsx.Service
    Platform platform.Service
    Update   update.Service
    Out      Output
}

type DoctorOptions struct {
    CheckUpdate bool
    ForceUpdate bool
}

func (d *Doctor) Run(ctx context.Context, cfg Config, opts DoctorOptions) error
```

## 19.3 Diseño del report

```go
type CheckStatus string

const (
    StatusOK   CheckStatus = "ok"
    StatusWarn CheckStatus = "warn"
    StatusErr  CheckStatus = "err"
)

type CheckResult struct {
    Name    string
    Status  CheckStatus
    Message string
    Hint    string
}
```

## 20. Diseño de Update Check

## 20.1 Estrategia cerrada

- Instalaciones por binario:
  - check contra fuente oficial de version/release
- Checkout de desarrollo:
  - git-based check como fallback o modo alternativo

## 20.2 Cache

Ubicación:

- macOS/Linux: `~/.agent47/cache/update.cache`
- Windows: `%LOCALAPPDATA%\agent47\cache\update.cache`

Modelo sugerido:

```go
type CacheRecord struct {
    CheckedAt    time.Time `json:"checked_at"`
    Status       string    `json:"status"`
    Method       string    `json:"method"`
    LocalVersion string    `json:"local_version"`
    LatestVersion string   `json:"latest_version"`
    Message      string    `json:"message"`
}
```

Nota:

Se puede usar JSON en el runtime Go aunque el shell actual use otro formato. El contrato importante es la semántica, no el encoding on-disk.

## 20.3 API sugerida

```go
type Service struct {
    Cache Cache
    HTTP  HTTPClient
    Git   GitClient
}

func (s *Service) Check(ctx context.Context, cfg Config, opts CheckOptions) (Result, error)
```

## 21. Diseño de Platform Layer

## 21.1 Requisito

El producto necesita resolver:

- home dir
- local app data
- user bin dir
- PATH hints
- clipboard
- diferencias de instalación

## 21.2 Interfaz sugerida

```go
type Service interface {
    OS() string
    IsWindows() bool
    HomeDir() (string, error)
    LocalAppDataDir() (string, error)
    EnsureUserPath(dir string) error
    PathContains(dir string) bool
    CopyToClipboard(text string) error
    LauncherTarget(cfg Config) string
    UserVisibleCommandPath(cfg Config) string
}
```

## 21.3 Implementaciones esperadas

- `common.go`
- `darwin.go`
- `linux.go`
- `windows.go`

## 22. Diseño de Filesystem Layer

## 22.1 Objetivo

Que el resto del producto no dependa de `os` + `filepath` + `rename` de forma dispersa.

## 22.2 API sugerida

```go
type Service interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm fs.FileMode) error
    WriteFileAtomic(path string, data []byte, perm fs.FileMode) error
    MkdirAll(path string) error
    Exists(path string) bool
    IsDir(path string) bool
    Remove(path string) error
    RemoveAll(path string) error
    Rename(oldPath, newPath string) error
    CopyFile(src, dst string) error
    CopyDir(src, dst string) error
    NewTransaction(workDir string) (Transaction, error)
}
```

## 22.3 Operaciones que requieren especial atención

- reemplazo atómico de archivo
- reemplazo staged de directorios
- backups
- rollback
- symlink o fallback en Unix/Windows

## 22.4 Regla de implementación

Toda operación destructiva o de reconciliación debe pasar por `fsx` y no implementarse ad hoc en los casos de uso.

## 23. Fases de Implementación

## Fase 0. Congelamiento funcional

### Objetivo

Formalizar el comportamiento observable actual para evitar regressions de producto.

### Trabajo concreto

- inventariar comandos
- inventariar flags
- inventariar outputs relevantes
- documentar exit codes
- documentar side effects por comando
- inventariar checks de `doctor`
- convertir install, doctor, bootstrap y update en matrices de contrato
- capturar cambios deliberados como la eliminación de `init-agent`
- separar explícitamente:
  - paridad obligatoria con el producto actual
  - cambios deliberados de producto aceptados en la migración

### Resultado esperado

Un documento interno de paridad tipo:

```text
command: afs add-agent
flags: --force, --only-skills
creates:
  - AGENTS.md
  - rules/*.yaml
  - skills/*
  - skills/AVAILABLE_SKILLS.xml
  - README.md (empty if missing)
preserves:
  - README.md
  - specs/spec.yml
  - SNAPSHOT.md
  - SPEC.md
```

La tabla de paridad debe existir para cada comando público e incluir:

- inputs/flags soportados
- side effects en filesystem
- output observable relevante
- exit codes
- warnings o degradaciones aceptadas
- semántica de mantenimiento:
  - install sin `--force`
  - install con `--force`
  - matching de `doctor`
  - cleanup legacy

### Criterio de salida

- suficientemente claro para escribir tests Go equivalentes

## Fase 1. Esqueleto Go

### Objetivo

Crear el nuevo entrypoint del producto y sus tipos base.

### Tareas concretas

- crear `go.mod`
- crear `cmd/afs/main.go`
- definir `Config`
- definir `Output`
- definir router mínimo
- implementar `help`
- implementar `version`

### Ejemplo mínimo

```go
func main() {
    cfg, err := runtime.Detect()
    if err != nil {
        fmt.Fprintf(os.Stderr, "[ERR] %v\n", err)
        os.Exit(1)
    }
    os.Exit(app.Run(context.Background(), cfg, os.Args[1:]))
}
```

### Criterio de salida

- compila en macOS/Linux/Windows

## Fase 2. Platform + FS primitives

### Objetivo

Construir las primitivas base portables.

### Tareas concretas

- `internal/platform`
- `internal/fsx`
- tests unitarios por operación crítica

### Criterio de salida

- hay primitives suficientes para install/bootstrap sin shell

## Fase 3. Templates + Manifest

### Objetivo

Resolver la carga de templates y el contrato declarativo.

### Tareas concretas

- embedded source
- filesystem source
- parser de manifest
- validator

### Criterio de salida

- el runtime Go puede responder qué assets existen y qué targets se gestionan

## Fase 4. Skills

### Objetivo

Migrar validación y XML generation.

### Tareas concretas

- parser de frontmatter
- validator
- discover
- XML

### Criterio de salida

- golden tests pasando

## Fase 5. Bootstrap

### Objetivo

Implementar `add-agent` y variantes.

### Tareas concretas

- staging
- backup
- commit
- rollback
- `--force`
- `--only-skills`

### Criterio de salida

- suite de paridad de bootstrap pasando en macOS/Linux/Windows

## Fase 6. Prompt helpers

### Objetivo

Migrar `add-agent-prompt` y `add-ss-prompt`.

### Criterio de salida

- ambos comandos existen en Go y respetan semántica actual

## Fase 7. Doctor + Update

### Objetivo

Migrar diagnóstico y update checks.

### Criterio de salida

- `doctor` funcional en las tres plataformas

## Fase 8. Install / Uninstall

### Objetivo

Hacer que la instalación real apunte al runtime Go.

### Tareas concretas

- reescribir `install.sh` como bootstrap del binario Go
- preservar `install.sh --non-interactive`
- implementar instalación Windows simple
- implementar uninstall
- actualizar docs de instalación

### Criterio de salida

- el usuario instala y usa el runtime Go, no el runtime shell

## Fase 9. Tests + CI

### Objetivo

Cambiar el centro de gravedad del testing a Go.

### Tareas concretas

- convertir tests Bats críticos
- mantener smoke tests
- CI matrix:
  - macOS
  - Linux
  - Windows

### Criterio de salida

- la validación principal ya no depende de Bats

## Fase 10. Default Runtime Switch

### Objetivo

Declarar oficialmente el runtime Go como implementación principal.

### Tareas concretas

- actualizar docs
- publicar release Go-first
- deprecar shell runtime
- mantener explícito en docs que root `SPEC.md` del repo sigue siendo una spec del estado actual del producto y no una feature spec de trabajo
- mantener documentada la eliminación de `init-agent` como cambio deliberado de producto

## Fase 11. Cleanup

### Objetivo

Eliminar deuda residual.

### Tareas concretas

- retirar shell code obsoleto
- reducir CI legacy
- simplificar repo

## 24. Testing Strategy

## 24.1 Unit tests

Cubrir:

- manifest parser
- manifest validation
- template source resolution
- frontmatter parsing
- skill validation
- XML generation
- update cache
- output formatting
- platform path helpers
- atomic file replacement

Ejemplo:

```go
func TestManifestValidateMissingSections(t *testing.T) {
    m := manifest.Manifest{}
    err := m.Validate()
    require.Error(t, err)
}
```

## 24.2 Integration tests

Cada test crea un home temporal o workdir temporal.

Cubrir:

- install macOS/Linux-like env
- uninstall
- add-agent fresh repo
- add-agent --force
- rollback induced failure
- doctor installed runtime

## 24.3 Golden tests

Usar `testdata/golden/`.

Cubrir:

- help
- doctor
- `AVAILABLE_SKILLS.xml`
- manifest-driven outputs

## 24.4 Cross-platform tests

CI debe correr:

```yaml
os:
  - ubuntu-latest
  - macos-latest
  - windows-latest
```

## 24.5 Smoke tests

Release smoke:

1. instalar runtime
2. correr `afs doctor`
3. correr `afs add-agent`
4. validar scaffold

## 25. CI/CD Strategy

## 25.1 CI mínima

- `go test ./...`
- smoke install
- scaffold smoke
- optional lint

## 25.2 Release artifacts

Publicar:

- darwin/amd64
- darwin/arm64
- linux/amd64
- linux/arm64
- windows/amd64
- windows/arm64 si el proyecto decide soportarlo oficialmente

## 25.3 Checksums

Generar:

```text
checksums.txt
```

## 25.4 Scripts de release

Se puede conservar shell en tooling de release, pero no en el runtime del producto.

## 26. Compatibilidad con Desarrollo desde GitHub

Requisito explícito del proyecto:

- seguir iterando cómodamente desde GitHub

Esto implica:

- mantener `templates/` como archivos normales del repo
- soportar modo desarrollo filesystem-based
- no ocultar todo detrás del binario embebido durante el desarrollo

Ejemplo de resolución:

```go
func DetectTemplateMode(env map[string]string) TemplateMode {
    if env["AGENT47_TEMPLATE_SOURCE"] == "filesystem" {
        return TemplateModeFilesystem
    }
    return TemplateModeEmbedded
}
```

## 27. Riesgos Técnicos Detallados

## 27.1 Atomic replace en Windows

Riesgo:

- el patrón Unix de `rename` puede no ser suficiente

Mitigación:

- centralizarlo en `fsx`
- testearlo con temp dirs en Windows CI

## 27.2 PATH mutation

Riesgo:

- modificar PATH del usuario es platform-sensitive

Mitigación:

- encapsular en `platform`
- dar fallback a hints claros si la automatización falla

## 27.3 Drift de output

Riesgo:

- el nuevo CLI cambie mensajes demasiado

Mitigación:

- golden tests
- revisar outputs de help/doctor/bootstrap

## 27.4 Drift de semántica de `--force`

Riesgo:

- perder reconciliación o preserved targets

Mitigación:

- suite dedicada de tests de bootstrap
- invariants explícitos

## 28. Invariants de Implementación

Estas reglas no deben romperse:

- `afs` sigue siendo el único comando público
- `install.sh --non-interactive` sigue existiendo
- helpers publicados siguen formando parte del contrato observable
- `README.md` vacío si falta
- `specs/spec.yml` no se scaffoldéa por defecto
- root `SPEC.md` sigue siendo documental y describe el estado actual del producto, no una feature spec nueva
- root `SPEC.md` se preserva durante refresh igual que `SNAPSHOT.md`
- `--force` reconcilia managed targets
- preserved targets no se tocan
- releases usan templates embebidos
- desarrollo local puede usar filesystem templates
- macOS sigue usando `install.sh`

## 29. Checklist de Implementación

## 29.1 Foundation

- [ ] crear `go.mod`
- [ ] crear `cmd/afs`
- [ ] definir `Config`
- [ ] definir `Output`
- [ ] implementar router mínimo

## 29.2 Platform/FS

- [ ] home dirs
- [ ] user bin dir
- [ ] local app data dir
- [ ] clipboard
- [ ] PATH helpers
- [ ] atomic file write
- [ ] staged dir replace
- [ ] rollback primitives

## 29.3 Templates/Manifest

- [ ] embedded source
- [ ] filesystem source
- [ ] loader
- [ ] parser de manifest
- [ ] validator

## 29.4 Skills

- [ ] frontmatter parser
- [ ] validator
- [ ] XML generation

## 29.5 Commands

- [ ] help
- [ ] doctor
- [ ] uninstall
- [ ] add-agent
- [ ] add-agent-prompt
- [ ] add-ss-prompt
- [ ] mantener documentada la eliminación de `init-agent` en la transición a Go

## 29.6 Install

- [ ] managed binary install
- [ ] macOS/Linux launcher
- [ ] Windows PATH integration
- [ ] uninstall flow

## 29.7 Tests

- [ ] unit tests
- [ ] integration tests
- [ ] golden tests
- [ ] CI matrix

## 30. Recomendación Final

La migración correcta para `agent47` es:

- conservadora en contrato
- agresiva en portabilidad interna
- explícita en compatibilidad macOS
- práctica en Windows
- progresiva en testing y distribución

El proyecto no necesita “parecer otro producto”. Necesita seguir siendo `agent47`, pero con un runtime Go portable, mantenible y publicable.
