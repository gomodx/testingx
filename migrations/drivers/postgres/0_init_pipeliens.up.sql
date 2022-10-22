create table sources
(
    id         text primary key not null,
    name       text             not null,
    type       text             not null,
    config     jsonb            not null,
    created_at timestamptz      not null default now(),
    updated_at timestamptz      not null default now(),
    deleted_at timestamptz
);

create table sinks
(
    id         text primary key not null,
    name       text             not null,
    type       text             not null,
    config     jsonb            not null,
    created_at timestamptz      not null default now(),
    updated_at timestamptz      not null default now(),
    deleted_at timestamptz
);

create table pipelines
(
    id         text primary key not null,
    name       text             not null,
    created_at timestamptz      not null default now(),
    updated_at timestamptz      not null default now(),
    deleted_at timestamptz
);

create table pipeline_edges
(
    pipeline_id text        not null,
    source_id   text        not null,
    sink_id     text        not null,
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),
    deleted_at  timestamptz,
    PRIMARY KEY (pipeline_id,source_id, sink_id)
);
