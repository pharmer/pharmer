DROP SCHEMA IF EXISTS "pharmer" CASCADE;

CREATE SCHEMA "pharmer" AUTHORIZATION "{$NS_USER}";
SET search_path TO "pharmer";

START TRANSACTION;
SET standard_conforming_strings=off;
SET escape_string_warning=off;
SET CONSTRAINTS ALL DEFERRED;

CREATE TABLE "credential" (
    "id" bigserial,
    "kind" text NOT NULL,
    "apiVersion" text NOT NULL,
    "name" text NOT NULL,
    "uid" text NOT NULL,
    "resourceVersion" text NOT NULL,
    "generation" bigint NOT NULL,
    "labels" text default '{}',
    "data" text NOT NULL,
    "creationTimestamp" bigint NOT NULL,
    "dateModified" bigint NOT NULL,
    "deletionTimestamp" bigint,
    PRIMARY KEY ("id"),
    UNIQUE ("name"),
    UNIQUE ("uid")
);

CREATE TABLE "cluster" (
    "id" bigserial,
    "kind" text NOT NULL,
    "apiVersion" text NOT NULL,
    "name" text NOT NULL,
    "uid" text NOT NULL,
    "resourceVersion" text NOT NULL,
    "generation" bigint NOT NULL,
    "labels" text default '{}',
    "data" text NOT NULL,
    "creationTimestamp" bigint NOT NULL,
    "dateModified" bigint NOT NULL,
    "deletionTimestamp" bigint,
    PRIMARY KEY ("id"),
    UNIQUE ("name"),
    UNIQUE ("uid")
);

CREATE TABLE "nodegroup" (
    "id" bigserial,
    "kind" text NOT NULL,
    "apiVersion" text NOT NULL,
    "name" text NOT NULL,
    "clusterName" text NOT NULL,
    "uid" text NOT NULL,
    "resourceVersion" text NOT NULL,
    "generation" bigint NOT NULL,
    "labels" text default '{}',
    "data" text NOT NULL,
    "creationTimestamp" bigint NOT NULL,
    "dateModified" bigint NOT NULL,
    "deletionTimestamp" bigint,
    PRIMARY KEY ("id"),
    UNIQUE ("name", "clusterName"),
    UNIQUE ("uid")
);

CREATE TABLE "certificate" (
    "id" bigserial,
    "name" text NOT NULL,
    "clusterName" text NOT NULL,
    "uid" text NOT NULL,
    "cert" text NOT NULL,
    "key" bigint NOT NULL,
    "creationTimestamp" bigint NOT NULL,
    "dateModified" bigint NOT NULL,
    "deletionTimestamp" bigint,
    PRIMARY KEY ("id"),
    UNIQUE ("name", "clusterName"),
    UNIQUE ("uid")
);

CREATE TABLE "sshKey" (
    "id" bigserial,
    "name" text NOT NULL,
    "clusterName" text NOT NULL,
    "uid" text NOT NULL,
    "publicKey" text NOT NULL,
    "privateKey" bigint NOT NULL,
    "creationTimestamp" bigint NOT NULL,
    "dateModified" bigint NOT NULL,
    "deletionTimestamp" bigint,
    PRIMARY KEY ("id"),
    UNIQUE ("name", "clusterName"),
    UNIQUE ("uid")
);

-- Owner-Alter-Table --
ALTER TABLE "credential" OWNER TO "{$NS_USER}";
ALTER TABLE "cluster" OWNER TO "{$NS_USER}";
ALTER TABLE "nodegroup" OWNER TO "{$NS_USER}";
ALTER TABLE "certificate" OWNER TO "{$NS_USER}";
ALTER TABLE "sshKey" OWNER TO "{$NS_USER}";

-- Post-data save --
COMMIT;
START TRANSACTION;

-- Typecasts --

-- Foreign keys --

-- Sequences --

-- Full Text keys --

COMMIT;