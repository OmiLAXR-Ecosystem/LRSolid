CREATE TABLE IF NOT EXISTS attachment_meta (
                                               tenant_id     text NOT NULL,
                                               sha256        text NOT NULL,

                                               content_type  text NOT NULL,
                                               length        bigint NOT NULL,
                                               file_url      text NULL,

                                               stored_bucket text NULL,
                                               stored_key    text NULL,
                                               stored_size   bigint NULL,
                                               stored_sha256 text NULL,

                                               created_at    timestamptz NOT NULL,

                                               PRIMARY KEY (tenant_id, sha256)
    );