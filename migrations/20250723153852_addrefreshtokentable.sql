-- +goose Up
-- +goose StatementBegin
create table refresh_token(
    refresh_token_id UUID primary key,
    user_id UUID not null references "user"(user_id) on delete cascade,
    token_hash varchar(255) not null,
    ip_address varchar(45),
    user_agent text,
    created_at timestamptz not null default current_timestamp,
    expires_at timestamptz not null
);

create index idx_refresh_token_user_id on refresh_token(user_id);
create index idx_refresh_token_hash on refresh_token(token_hash);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index idx_refresh_token_hash;
drop index idx_refresh_token_user_id;
drop table refresh_token;
-- +goose StatementEnd
