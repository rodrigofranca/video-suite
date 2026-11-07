# PRD Padrao - UltraMedia Platform

**Status**: Draft  
**Fonte de discovery**: `brainstorming.md`  
**Produto**: UltraMedia Platform / Open Video Ops  
**Horizonte inicial**: MVP 1 - Video Processing Core

## 1. Executive Summary

### Problem Statement

Criadores, engenheiros de video e times tecnicos que lidam com midia precisam combinar varias ferramentas isoladas para transformar videos brutos em assets prontos para distribuicao. Hoje, tarefas como upload, extracao de metadata, transcoding, thumbnails, HLS, logs, preview e futuras automacoes de IA tendem a ficar espalhadas entre scripts, servicos externos e dashboards pouco integrados.

### Proposed Solution

Construir uma plataforma open source, modular e AI-first para ingestao, processamento, enriquecimento, operacao e distribuicao de videos. O MVP inicial deve provar o core de Video Ops: upload de MP4, criacao de media asset, extracao de metadata com FFmpeg/ffprobe, geracao de thumbnail, transcode para HLS, preview no dashboard e visualizacao de status/logs dos jobs.

### Business Impact

- Posicionar o projeto como uma referencia pratica em video engineering open source, indo alem de player ou encoder isolado.
- Criar uma base tecnica reutilizavel para features de maior valor: AI metadata, clipping automatico, NOC com IA e live streaming.
- Gerar portfolio tecnico demonstravel para oportunidades em video engineering, media infrastructure e plataformas AI-first.

### Timeline

| Marco | Prazo alvo | Resultado |
|---|---:|---|
| Foundation design | Semana 1 | Modelo de asset, contratos de job e arquitetura do MVP definidos |
| MVP 1 implementation | Semanas 2-6 | Upload, metadata, thumbnail, HLS, preview e logs funcionando |
| Hardening | Semanas 7-8 | Erros, retries, UX basica e documentacao inicial |
| Next MVP planning | Semana 8 | PRD tecnico para AI Metadata Pipeline |

### Resources Required

- 1 product/engineering owner.
- 1 full-stack engineer com SvelteKit + Go.
- 1 engineer/worker specialist para FFmpeg, storage e filas.
- Infra local/dev: PostgreSQL, Redis ou NATS, MinIO/S3-compatible, FFmpeg.
- Budget inicial: open source/self-hosted; custo principal e tempo de desenvolvimento e computacao de processamento.

### Success Metrics

- 90%+ dos uploads validos completam metadata extraction e thumbnail generation em ambiente local/dev.
- 80%+ dos videos MP4 de teste geram HLS reproduzivel no dashboard.
- P95 de inicio de job abaixo de 10 segundos em ambiente local com fila vazia.
- 100% dos jobs exibem status, progresso basico, logs e artefatos gerados.
- Primeiro usuario tecnico consegue rodar o fluxo com documentacao em menos de 30 minutos.

## 2. Problem Definition

### 2.1 Customer Problem

- **Who**: Engenheiros de video, desenvolvedores full-stack que trabalham com media, criadores tecnicos, times pequenos de streaming/educacao/comunidade e builders que precisam de pipelines de video.
- **What**: Dificuldade para transformar video bruto em asset pronto para playback, analise e automacao sem montar uma cadeia propria de scripts e ferramentas desconectadas.
- **When**: Sempre que um video precisa ser ingerido, inspecionado, processado, convertido, enriquecido ou monitorado.
- **Where**: Ambientes de desenvolvimento, pipelines internos, plataformas de conteudo, ferramentas de criador e setups self-hosted.
- **Why**: O dominio de video combina storage, jobs longos, FFmpeg, manifests, metadata, AI, logs e delivery. Essas responsabilidades frequentemente nao compartilham um modelo comum de asset/job.
- **Impact**: Sem uma plataforma integrada, o time perde velocidade, observabilidade, padronizacao e capacidade de evoluir para IA, clipping ou live sem reescrever a base.

### 2.2 Market Opportunity

- **Market Size**: A validar. O espaco inclui video infrastructure, creator tools, streaming tooling, media asset management leve e AI video workflows.
- **Growth Rate**: A validar com pesquisa de mercado. A tese e que demanda cresce com video curto, automacao por IA, educacao online, comunidades e conteudo B2B.
- **Competition**: Solucoes comerciais como Mux, Cloudinary, AWS MediaConvert, Bitmovin, Vimeo, Descript e ferramentas de clipping/AI. A lacuna inicial e uma plataforma open source modular, educacional e self-hosted para Video Ops.
- **Timing**: FFmpeg, modelos de transcricao, storage S3-compatible e AI workflows estao maduros o suficiente para compor uma plataforma aberta com boa experiencia de desenvolvimento.

### 2.3 Business Case

- **Revenue Potential**: Inicialmente portfolio/open source. Futuro potencial em cloud hosted, consultoria, plugins premium, templates enterprise ou suporte gerenciado.
- **Cost Savings**: Reduz tempo de criar pipelines internos de video e evita dependencia precoce de servicos pagos para fluxos basicos.
- **Strategic Value**: Cria uma base tecnica forte para demonstrar expertise em video engineering, AI workflows, observabilidade e produto open source.
- **Risk Assessment**: Se o projeto comecar por live, clipping ou plugins avancados, ha risco alto de arquitetura excessiva antes de provar o core.

## 3. Solution Overview

### 3.1 Proposed Solution

Construir a UltraMedia Platform como uma Video Ops Suite open source em monorepo modular. O produto comeca por um core de VOD processing e evolui por camadas: Media Core, Processing Engine, AI Intelligence Layer, Delivery Layer, Operations/NOC e Live Layer.

#### Key Capabilities

- Media asset core para representar videos, fontes, renditions, tracks, subtitles, thumbnails, chapters, cuepoints, clips e metadata.
- Processing engine com jobs, workers, retries, logs, artefatos, status e progresso.
- FFmpeg processing para metadata, thumbnails e HLS.
- Dashboard para upload, listagem de assets, preview, jobs e logs.
- Base extensivel para AI metadata, clipping, NOC e live.

#### User Journey

```txt
Upload MP4
  -> create media asset
  -> extract technical metadata
  -> generate thumbnail
  -> transcode/package HLS
  -> preview playback
  -> inspect job logs/status
```

#### Differentiation

- Open source e self-hosted por design.
- Foco em Video Ops modular, nao apenas player, encoder ou ferramenta de clipping.
- Core orientado a jobs que permite evoluir para runners locais, containers, workers HTTP, filas, servicos externos e plugins futuros.
- AI-first de forma incremental: IA entra sobre uma base solida de assets, jobs, logs e artefatos.

### 3.2 In Scope

- **P0 - Asset management**: criar, listar e visualizar media assets.
- **P0 - Upload MP4**: aceitar upload de video e persistir source original.
- **P0 - Job engine basico**: criar jobs, executar worker, registrar status, progresso, logs e erros.
- **P0 - Metadata extraction**: usar ffprobe para extrair duracao, codec, resolucao, bitrate e streams.
- **P0 - Thumbnail generation**: gerar ao menos uma thumbnail por asset.
- **P0 - HLS encoding**: gerar manifest e segmentos HLS basicos a partir do MP4.
- **P0 - Dashboard preview**: reproduzir HLS no dashboard.
- **P1 - Storage abstraction**: suportar MinIO/S3-compatible no design inicial.
- **P1 - Job logs UI**: exibir logs e artefatos por job.
- **P2 - Documentacao de setup**: guia para rodar localmente.

### 3.3 Out of Scope

- Live streaming, RTMP/SRT/WebRTC ingest e restreaming.
- Clipping automatico e export vertical.
- Transcricao, subtitles automaticas, chapters por IA e tags por IA.
- Sistema de plugins completo.
- Billing, multi-tenant enterprise, RBAC avancado ou marketplace.
- CDN production-grade, DRM, SSAI/ads e QoE analytics completo.
- NOC com IA e explicacao automatica de incidentes.

### 3.4 MVP Definition

- **Core Features**: upload, asset, metadata extraction, thumbnail, HLS, preview, job status e logs.
- **Success Criteria**: um usuario tecnico consegue enviar um MP4, acompanhar jobs e assistir ao HLS resultante no dashboard.
- **Timeline**: 6 a 8 semanas para MVP funcional.
- **Learning Goals**: validar modelo de asset, contrato de jobs, integracao FFmpeg, arquitetura modular monolith + workers e fluxo basico de Video Ops.

## 4. User Stories & Requirements

### 4.1 User Stories

```txt
As a video engineer
I want to upload a video and see its processing pipeline
So that I can validate the platform core and inspect generated assets.

Acceptance Criteria:
- [ ] Upload creates a media asset.
- [ ] Metadata extraction job runs automatically.
- [ ] Thumbnail generation job runs automatically.
- [ ] HLS encoding job can be triggered or runs automatically by pipeline.
- [ ] Dashboard shows status, logs, metadata, thumbnail and playback preview.
```

```txt
As a technical creator
I want to preview the generated HLS output in the browser
So that I can confirm the video is ready for delivery.

Acceptance Criteria:
- [ ] Asset detail page exposes generated HLS manifest.
- [ ] Player renders the HLS output.
- [ ] UI shows failed state when HLS generation fails.
```

```txt
As a platform maintainer
I want every processing task to run as a typed job
So that future features can reuse the same execution, retry and logging model.

Acceptance Criteria:
- [ ] Jobs have type, assetId, input, status, logs and timestamps.
- [ ] Worker can pick pending jobs and update status.
- [ ] Failures preserve error message and logs.
```

### 4.2 Functional Requirements

| ID | Requirement | Priority | Notes |
|---|---|---|---|
| FR1 | User can upload an MP4 file through the dashboard | P0 | First entry point for MVP |
| FR2 | System creates a media asset for each uploaded source | P0 | Foundation for all modules |
| FR3 | System stores original video in local/S3-compatible storage | P0 | MinIO recommended for local dev |
| FR4 | System extracts technical metadata using ffprobe | P0 | Duration, streams, codecs, resolution, bitrate |
| FR5 | System generates at least one thumbnail using FFmpeg | P0 | Used by dashboard and preview |
| FR6 | System transcodes/packages uploaded MP4 into HLS | P0 | Basic VOD HLS output |
| FR7 | User can preview generated HLS in dashboard | P0 | Connects pipeline to playback value |
| FR8 | System records job status, timestamps, logs and error messages | P0 | Required for observability |
| FR9 | User can inspect asset-level jobs and artifacts | P1 | Improves debugging |
| FR10 | System exposes API endpoints for assets, upload, jobs and artifacts | P1 | Enables dashboard and future automation |
| FR11 | System supports a stable job contract for future job types | P1 | Foundation for AI and plugins |
| FR12 | Documentation explains local setup and MVP flow | P2 | Required for open source adoption |

### 4.3 Non-Functional Requirements

- **Performance**: job creation should be near-real-time; processing duration depends on input size and machine capacity.
- **Scalability**: architecture must allow multiple workers later, but MVP can run as modular monolith + worker.
- **Security**: validate upload type/size, isolate storage paths, avoid shell injection in FFmpeg commands.
- **Reliability**: failed jobs must be visible, retryable in future, and never leave asset state ambiguous.
- **Usability**: dashboard should make pipeline state obvious without requiring terminal logs.
- **Portability**: local development should work with open source dependencies: PostgreSQL, Redis/NATS, MinIO and FFmpeg.
- **Observability**: every job should emit structured status, timestamps and logs; future OpenTelemetry should fit the model.

## 5. Design & User Experience

### 5.1 Design Principles

- **Operational clarity**: users should always know what happened to an asset, what is running and what failed.
- **Video-first workflow**: primary screens revolve around assets, processing state, generated media and playback.
- **Progressive complexity**: MVP keeps simple paths visible; advanced logs/artifacts are available when needed.
- **Developer-grade transparency**: expose metadata, manifests, FFmpeg logs and artifacts rather than hiding internals.

### 5.2 Wireframes/Mockups

Initial screens to design:

- Asset list with thumbnail, filename, duration, processing status and latest job state.
- Upload screen with file validation and progress.
- Asset detail page with metadata panel, thumbnail, HLS preview and artifact links.
- Jobs panel with job timeline, status, logs and error details.

### 5.3 Information Architecture

```txt
Dashboard
  Assets
    Asset Detail
      Overview
      Metadata
      Playback
      Jobs
      Artifacts
  Jobs
    Job Detail
  Settings
    Storage
    Workers
```

## 6. Technical Specifications

### 6.1 Architecture Overview

Recommended initial architecture:

```txt
SvelteKit Dashboard
  -> Go API
    -> PostgreSQL
    -> Redis/NATS queue
    -> MinIO/S3-compatible storage
    -> Worker
      -> FFmpeg/ffprobe
      -> artifacts storage
```

#### Stack

| Area | Initial choice |
|---|---|
| Frontend | SvelteKit |
| API | Go |
| Workers | Go for core processing; Python later for AI workers |
| Processing | FFmpeg + ffprobe |
| Database | PostgreSQL |
| Queue | Redis or NATS |
| Storage | MinIO/S3-compatible |
| Observability | Structured logs first; OpenTelemetry later |
| Player | UltraMedia / Web Component or HLS-compatible player |

### 6.2 API Design

Initial API surface:

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/assets/upload` | Upload video and create asset |
| `GET` | `/assets` | List assets |
| `GET` | `/assets/{id}` | Fetch asset detail |
| `GET` | `/assets/{id}/jobs` | List jobs for asset |
| `POST` | `/assets/{id}/jobs` | Trigger processing job |
| `GET` | `/jobs/{id}` | Fetch job detail |
| `GET` | `/assets/{id}/artifacts` | List generated artifacts |

Authentication can be omitted in the local MVP, but API boundaries should not assume anonymous production use.

### 6.3 Database Design

Core entities:

- `media_assets`: id, source_id, filename, status, duration, created_at, updated_at.
- `media_sources`: id, asset_id, storage_key, mime_type, size_bytes, checksum.
- `processing_jobs`: id, asset_id, type, input_json, status, progress, error_message, created_at, started_at, completed_at.
- `job_logs`: id, job_id, level, message, timestamp.
- `artifacts`: id, asset_id, job_id, type, storage_key, metadata_json, created_at.
- `technical_metadata`: id, asset_id, metadata_json, extracted_at.

Migration strategy should prioritize stable IDs and explicit relationships between asset, job and artifact.

### 6.4 Security Considerations

- Validate MIME type, extension and file size before processing.
- Store uploads with generated keys, not user-provided paths.
- Execute FFmpeg with argument arrays, not interpolated shell strings.
- Restrict artifact access through signed URLs or controlled API paths.
- Avoid logging secrets, storage credentials or signed URLs.
- Add auth/RBAC before any hosted or multi-user deployment.

## 7. Go-to-Market Strategy

### 7.1 Launch Plan

- **Soft Launch**: publish MVP as developer preview with local setup docs, demo video and example MP4.
- **Full Launch**: release open source beta after HLS preview, logs and setup stability are reliable.
- **Marketing**: technical article series on Video Ops, FFmpeg pipeline, HLS packaging, job engine and future AI workflows.
- **Support**: README, architecture docs, troubleshooting guide and issue templates.

### 7.2 Pricing Strategy

Initial project is open source. Future options:

- Hosted cloud version.
- Paid support/consulting.
- Premium templates or enterprise integrations.
- Managed processing workers.

### 7.3 Success Metrics

| Metric | Target | Measurement Method |
|---|---:|---|
| Local setup success | < 30 min | Fresh machine/dev walkthrough |
| Upload-to-preview completion | 80%+ valid test videos | Automated/manual test suite |
| Job visibility coverage | 100% jobs visible | Dashboard + API checks |
| Processing failure diagnosability | 100% failed jobs with logs/error | Job records |
| Open source traction | To define | Stars, forks, issues, contributors |

## 8. Risks & Mitigations

| Risk | Probability | Impact | Mitigation Strategy |
|---|---|---|---|
| Scope creep into live/AI/clipping too early | High | High | Keep MVP restricted to VOD core and document out-of-scope |
| FFmpeg failures vary by input file | High | Medium | Use controlled sample set, preserve logs, validate inputs |
| Architecture overbuild around plugins | Medium | High | Start with job contract; defer plugin system |
| Local setup becomes complex | Medium | High | Provide Docker Compose and minimal seed/demo flow |
| Processing large files consumes too many resources | Medium | Medium | Define MVP file-size limits and document hardware assumptions |
| Weak asset model blocks future modules | Medium | High | Design Media Core before adding advanced features |
| Dashboard hides technical state | Low | Medium | Make logs, artifacts and metadata first-class in UI |

## 9. Timeline & Milestones

| Milestone | Date | Deliverables | Success Criteria |
|---|---|---|---|
| Product/technical design | Week 1 | Asset model, job contract, architecture notes | Scope approved for MVP 1 |
| Upload + asset core | Week 2 | Upload API, storage, asset list/detail | MP4 creates asset and source |
| Metadata + thumbnail jobs | Week 3 | ffprobe extraction, thumbnail artifact | Dashboard shows metadata and thumbnail |
| HLS encoding | Weeks 4-5 | HLS job, manifest/segments, artifact records | Generated HLS plays locally |
| Job observability | Week 6 | Logs, status, progress, failure states | Failed and successful jobs inspectable |
| MVP hardening | Weeks 7-8 | Docs, sample videos, bug fixes, demo | Fresh setup completes upload-to-preview |

## 10. Team & Resources

### 10.1 Team Structure

- **Product/Engineering Owner**: defines scope, PRD, roadmap and open source positioning.
- **Engineering Lead**: owns architecture, API, asset model and job contract.
- **Frontend Engineer**: owns SvelteKit dashboard and playback experience.
- **Worker/Media Engineer**: owns FFmpeg integration, processing jobs and artifacts.
- **QA/DevEx Owner**: can be shared; owns setup docs, sample files and validation flows.

### 10.2 Budget

- Development: primarily engineering time.
- Infrastructure: local/dev open source stack; future hosted demos may require cloud storage/compute.
- Marketing: technical writing, demo recording and documentation.
- Total: to be defined after confirming team/time availability.

## 11. Appendix

### Source Discovery Summary

The brainstorming positions the product as:

> Uma Video Ops Suite open source para transformar videos brutos em assets prontos para distribuicao, analise e automacao.

Recommended sequencing:

1. MVP 1 - Video Processing Core.
2. MVP 2 - AI Metadata Pipeline.
3. MVP 3 - AI Clip Generator.
4. NOC open source com IA.
5. Live streaming.

### Dependency Map

| Feature | Depends on | Unlocks |
|---|---|---|
| Asset management | Database, storage | Everything |
| Metadata extraction | Asset management | Search, AI, debugging |
| Job engine | Asset management | Encoding, AI, workers |
| HLS encoding | Job engine, FFmpeg | Playback, delivery |
| Thumbnail generation | Job engine, FFmpeg | Dashboard, preview |
| Transcription | Job engine, AI runner | Subtitles, chapters, clips |
| Scene detection | Job engine, AI/CV | Clipping, chapters |
| NOC dashboard | Jobs, logs, metrics | AI ops |
| Live ingest | Delivery, monitoring | Live streaming |

### Open Questions

1. O MVP deve usar Redis ou NATS como fila inicial?
2. O dashboard deve ser parte do mesmo app SvelteKit ou separado por pacote desde o inicio?
3. O HLS inicial deve gerar renditions multiplas ou apenas uma rendition funcional?
4. Qual limite de tamanho/duracao de video sera suportado no MVP local?
5. O player inicial sera UltraMedia, hls.js ou outro wrapper temporario?
6. Qual nivel de Docker Compose sera esperado para onboarding open source?
7. O projeto sera apresentado como `UltraMedia Platform`, `Open Video Ops` ou outro nome?
