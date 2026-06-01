
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "users" ("id" bigserial primary key,username varchar(255) NOT NULL UNIQUE,hash varchar(255),api_key varchar(255) NOT NULL UNIQUE );
CREATE TABLE IF NOT EXISTS "templates" ("id" bigserial primary key,user_id bigint,name varchar(255),subject varchar(255),text text,html text,modified_date timestamp with time zone );
CREATE TABLE IF NOT EXISTS "targets" ("id" bigserial primary key,first_name varchar(255),last_name varchar(255),email varchar(255),position varchar(255) );
CREATE TABLE IF NOT EXISTS "smtp" ("smtp_id" bigserial primary key,campaign_id bigint,host varchar(255),username varchar(255),from_address varchar(255) );
CREATE TABLE IF NOT EXISTS "results" ("id" bigserial primary key,campaign_id bigint,user_id bigint,r_id varchar(255),email varchar(255),first_name varchar(255),last_name varchar(255),status varchar(255) NOT NULL ,ip varchar(255),latitude real,longitude real );
CREATE TABLE IF NOT EXISTS "pages" ("id" bigserial primary key,user_id bigint,name varchar(255),html text,modified_date timestamp with time zone );
CREATE TABLE IF NOT EXISTS "groups" ("id" bigserial primary key,user_id bigint,name varchar(255),modified_date timestamp with time zone );
CREATE TABLE IF NOT EXISTS "group_targets" (group_id bigint,target_id bigint );
CREATE TABLE IF NOT EXISTS "events" ("id" bigserial primary key,campaign_id bigint,email varchar(255),time timestamp with time zone,message varchar(255) );
CREATE TABLE IF NOT EXISTS "campaigns" ("id" bigserial primary key,user_id bigint,name varchar(255) NOT NULL ,created_date timestamp with time zone,completed_date timestamp with time zone,template_id bigint,page_id bigint,status varchar(255),url varchar(255) );
CREATE TABLE IF NOT EXISTS "attachments" ("id" bigserial primary key,template_id bigint,content text,type varchar(255),name varchar(255) );

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "attachments";
DROP TABLE IF EXISTS "campaigns";
DROP TABLE IF EXISTS "events";
DROP TABLE IF EXISTS "group_targets";
DROP TABLE IF EXISTS "groups";
DROP TABLE IF EXISTS "pages";
DROP TABLE IF EXISTS "results";
DROP TABLE IF EXISTS "smtp";
DROP TABLE IF EXISTS "targets";
DROP TABLE IF EXISTS "templates";
DROP TABLE IF EXISTS "users";
