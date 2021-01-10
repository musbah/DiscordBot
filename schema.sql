CREATE TABLE users
(
    user_id bigint PRIMARY KEY,
    level integer,
    exp integer,
    max_hp integer,
    current_hp integer,
    max_mp     integer,
    current_mp integer,
    str       integer,
    agi       integer,
    dex       integer,
    def       integer
)