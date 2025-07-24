-- +goose Up
-- +goose StatementBegin
create table "user"(
    user_id uuid primary key
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table "user";
-- +goose StatementEnd
