-- +goose Up
-- +goose StatementBegin
create table token_black_list (
    token_id uuid primary key
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table token_black_list;
-- +goose StatementEnd
