-- +goose Up
-- +goose StatementBegin
alter table token_black_list add column revoke_at timestamptz not null;
comment on column token_black_list.revoke_at is
'The time, when the token should be revoked. After that time we could delete token from database';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table token_black_list drop column revoke_at;
-- +goose StatementEnd
