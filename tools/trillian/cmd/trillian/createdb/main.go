// Copyright 2022 The Sigstore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"database/sql"

	"chainguard.dev/exitdir"

	_ "github.com/go-sql-driver/mysql"

	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

// These are from the Trillian schema, with the comments removed.
// https://github.com/google/trillian/blob/master/storage/mysql/schema/storage.sql
const (
	// This is used to query to see if there are indices on a particular table.
	indexCheck = `
	select count(*) from information_schema.statistics where table_schema = ? and table_name=? and index_name=?;
	`
	createTableTrees = `
	CREATE TABLE IF NOT EXISTS Trees(
	  TreeId                BIGINT NOT NULL,
	  TreeState             ENUM('ACTIVE', 'FROZEN', 'DRAINING') NOT NULL,
	  TreeType              ENUM('LOG', 'MAP', 'PREORDERED_LOG') NOT NULL,
	  HashStrategy          ENUM('RFC6962_SHA256', 'TEST_MAP_HASHER', 'OBJECT_RFC6962_SHA256', 'CONIKS_SHA512_256', 'CONIKS_SHA256') NOT NULL,
	  HashAlgorithm         ENUM('SHA256') NOT NULL,
	  SignatureAlgorithm    ENUM('ECDSA', 'RSA', 'ED25519') NOT NULL,
	  DisplayName           VARCHAR(20),
	  Description           VARCHAR(200),
	  CreateTimeMillis      BIGINT NOT NULL,
	  UpdateTimeMillis      BIGINT NOT NULL,
	  MaxRootDurationMillis BIGINT NOT NULL,
	  PrivateKey            MEDIUMBLOB NOT NULL,
	  PublicKey             MEDIUMBLOB NOT NULL,
	  Deleted               BOOLEAN,
	  DeleteTimeMillis      BIGINT,
	  PRIMARY KEY(TreeId)
	);
`

	createTableTreeControl = `
	CREATE TABLE IF NOT EXISTS TreeControl(
	  TreeId                  BIGINT NOT NULL,
	  SigningEnabled          BOOLEAN NOT NULL,
	  SequencingEnabled       BOOLEAN NOT NULL,
	  SequenceIntervalSeconds INTEGER NOT NULL,
	  PRIMARY KEY(TreeId),
	  FOREIGN KEY(TreeId) REFERENCES Trees(TreeId) ON DELETE CASCADE
	);
`

	createTableSubtree = `
CREATE TABLE IF NOT EXISTS Subtree(
	  TreeId               BIGINT NOT NULL,
	  SubtreeId            VARBINARY(255) NOT NULL,
	  Nodes                MEDIUMBLOB NOT NULL,
	  SubtreeRevision      INTEGER NOT NULL,
	  -- Key columns must be in ASC order in order to benefit from group-by/min-max
	  -- optimization in MySQL.
	  PRIMARY KEY(TreeId, SubtreeId, SubtreeRevision),
	  FOREIGN KEY(TreeId) REFERENCES Trees(TreeId) ON DELETE CASCADE
	);
`

	createTableTreeHead = `
CREATE TABLE IF NOT EXISTS TreeHead(
	  TreeId               BIGINT NOT NULL,
	  TreeHeadTimestamp    BIGINT,
	  TreeSize             BIGINT,
	  RootHash             VARBINARY(255) NOT NULL,
	  RootSignature        VARBINARY(1024) NOT NULL,
	  TreeRevision         BIGINT,
	  PRIMARY KEY(TreeId, TreeHeadTimestamp),
	  FOREIGN KEY(TreeId) REFERENCES Trees(TreeId) ON DELETE CASCADE
	);
`

	createIndexTreeHeadRevision = `
	CREATE UNIQUE INDEX TreeHeadRevisionIdx
	  ON TreeHead(TreeId, TreeRevision);
`

	createTableLeafData = `
	CREATE TABLE IF NOT EXISTS LeafData(
	  TreeId               BIGINT NOT NULL,
	  -- This is a personality specific has of some subset of the leaf data.
	  -- It's only purpose is to allow Trillian to identify duplicate entries in
	  -- the context of the personality.
	  LeafIdentityHash     VARBINARY(255) NOT NULL,
	  -- This is the data stored in the leaf for example in CT it contains a DER encoded
	  -- X.509 certificate but is application dependent
	  LeafValue            LONGBLOB NOT NULL,
	  -- This is extra data that the application can associate with the leaf should it wish to.
	  -- This data is not included in signing and hashing.
	  ExtraData            LONGBLOB,
	  -- The timestamp from when this leaf data was first queued for inclusion.
	  QueueTimestampNanos  BIGINT NOT NULL,
	  PRIMARY KEY(TreeId, LeafIdentityHash),
	  FOREIGN KEY(TreeId) REFERENCES Trees(TreeId) ON DELETE CASCADE
	);
`

	createTableSequencedLeafData = `
CREATE TABLE IF NOT EXISTS SequencedLeafData(
	  TreeId               BIGINT NOT NULL,
	  SequenceNumber       BIGINT UNSIGNED NOT NULL,
	  -- This is a personality specific has of some subset of the leaf data.
	  -- It's only purpose is to allow Trillian to identify duplicate entries in
	  -- the context of the personality.
	  LeafIdentityHash     VARBINARY(255) NOT NULL,
	  -- This is a MerkleLeafHash as defined by the treehasher that the log uses. For example for
	  -- CT this hash will include the leaf prefix byte as well as the leaf data.
	  MerkleLeafHash       VARBINARY(255) NOT NULL,
	  IntegrateTimestampNanos BIGINT NOT NULL,
	  PRIMARY KEY(TreeId, SequenceNumber),
	  FOREIGN KEY(TreeId) REFERENCES Trees(TreeId) ON DELETE CASCADE,
	  FOREIGN KEY(TreeId, LeafIdentityHash) REFERENCES LeafData(TreeId, LeafIdentityHash) ON DELETE CASCADE
	);
`

	createIndexSequencedLeafMerkle = `
	CREATE INDEX SequencedLeafMerkleIdx
	  ON SequencedLeafData(TreeId, MerkleLeafHash);
`

	createTableUnsequenced = `
	CREATE TABLE IF NOT EXISTS Unsequenced(
	  TreeId               BIGINT NOT NULL,
	  -- The bucket field is to allow the use of time based ring bucketed schemes if desired. If
	  -- unused this should be set to zero for all entries.
	  Bucket               INTEGER NOT NULL,
	  -- This is a personality specific hash of some subset of the leaf data.
	  -- It's only purpose is to allow Trillian to identify duplicate entries in
	  -- the context of the personality.
	  LeafIdentityHash     VARBINARY(255) NOT NULL,
	  -- This is a MerkleLeafHash as defined by the treehasher that the log uses. For example for
	  -- CT this hash will include the leaf prefix byte as well as the leaf data.
	  MerkleLeafHash       VARBINARY(255) NOT NULL,
	  QueueTimestampNanos  BIGINT NOT NULL,
	  -- This is a SHA256 hash of the TreeID, LeafIdentityHash and QueueTimestampNanos. It is used
	  -- for batched deletes from the table when trillian_log_server and trillian_log_signer are
	  -- built with the batched_queue tag.
	  QueueID VARBINARY(32) DEFAULT NULL UNIQUE,
	  PRIMARY KEY (TreeId, Bucket, QueueTimestampNanos, LeafIdentityHash)
	);
`
)

// We need to create the tables in certain order because other tables
// depend on others, so we list the order here and can't just yolo the
// createTables map.
var tables = []string{
	"Trees",
	"TreeControl",
	"Subtree",
	"TreeHead",
	"LeafData",
	"SequencedLeafData",
	"Unsequenced",
}

// Map from table name to a statement that creates the table.
var createTables = map[string]string{
	"Trees":             createTableTrees,
	"TreeControl":       createTableTreeControl,
	"Subtree":           createTableSubtree,
	"TreeHead":          createTableTreeHead,
	"LeafData":          createTableLeafData,
	"SequencedLeafData": createTableSequencedLeafData,
	"Unsequenced":       createTableUnsequenced,
}

type indexCreate struct {
	tableName string
	createStr string
}

// Map from index name to table that should have it, which has statement
// to create the index if it's missing.
var createIndices = map[string]indexCreate{
	"TreeHeadRevisionIdx":    {"TreeHead", createIndexTreeHeadRevision},
	"SequencedLeafMerkleIdx": {"SequencedLeafData", createIndexSequencedLeafMerkle},
}

var (
	dbName   = flag.String("db_name", "trillian", "Database name to tack on to the connection string to select the right db.")
	mysqlURI = flag.String("mysql_uri", "", "SQL connection string in mysql format, for example: $(USER):$(PWD)@tcp($(HOST):3306)/$(DATABASE_NAME)")
)

func main() {
	// Signal via exitdir we are finished.
	defer func() {
		_ = exitdir.Exit()
	}()

	flag.Parse()
	if *mysqlURI == "" {
		log.Panicf("Need to specify mysql_uri to know where to connect to")
	}
	if *dbName == "" {
		log.Panicf("Need to specify database name")
	}

	connStr := fmt.Sprintf("%s/%s", strings.TrimSuffix(*mysqlURI, "/"), *dbName)
	ctx := signals.NewContext()
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Panicf("failed to open db connection: %v", err)
	}
	defer db.Close()
	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			logging.FromContext(ctx).Infof("Ping to DB succeeded")
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err := db.Ping(); err != nil {
		log.Panicf("failed to ping db: %v", err)
	}
	// Grab the tables
	existingTables := map[string]bool{}
	tableRows, err := db.Query("show tables")
	if err != nil {
		logging.FromContext(ctx).Panicf("show tables failed: %+v", err)
	}
	defer tableRows.Close()
	for next := tableRows.Next(); next; next = tableRows.Next() {
		var tableName string
		err = tableRows.Scan(&tableName)
		if err != nil {
			logging.FromContext(ctx).Errorf("Failed to get row %+v", err)
		}
		existingTables[tableName] = true
	}

	// Check the tables for existence and if they don't exist, create them.
	for _, table := range tables {
		if existingTables[table] {
			logging.FromContext(ctx).Infof("Table %q exists", table)
			continue
		}
		logging.FromContext(ctx).Infof("Table %q does not exist, creating", table)
		if _, err = db.Exec(createTables[table]); err != nil {
			logging.FromContext(ctx).Errorf("Failed to create table %q: %v", table, err)
		} else {
			logging.FromContext(ctx).Errorf("Created table %q", table)
		}
	}

	for indexName, tableAndCreate := range createIndices {
		tableName := tableAndCreate.tableName
		indexExists, err := indexExists(ctx, db, *dbName, indexName, tableName)
		if err != nil {
			logging.FromContext(ctx).Panicf("Failed to check %q on %q for existence", indexName, tableName)
		}
		if indexExists {
			logging.FromContext(ctx).Infof("Index %q exists on %q", indexName, tableName)
			continue
		}
		logging.FromContext(ctx).Infof("Index %q does not exist on %q, creating", indexName, tableName)
		if _, err = db.Exec(tableAndCreate.createStr); err != nil {
			logging.FromContext(ctx).Errorf("Failed to create index %q on %q: %v", indexName, tableName, err)
		} else {
			logging.FromContext(ctx).Errorf("Created index %q on table %q", indexName, tableName)
		}
	}
}

func indexExists(ctx context.Context, db *sql.DB, dbName, indexName, table string) (bool, error) {
	tableRows, err := db.Query(indexCheck, dbName, table, indexName)
	if err != nil {
		logging.FromContext(ctx).Panicf("checking for index failed: %+v", err)
		return false, err
	}
	defer tableRows.Close()
	var indexCount int64
	for next := tableRows.Next(); next; next = tableRows.Next() {
		err = tableRows.Scan(&indexCount)
		if err != nil {
			logging.FromContext(ctx).Errorf("Failed to get row %+v", err)
		}
		logging.FromContext(ctx).Infof("Found index %q on table %q : %+v", indexName, table, indexCount)
	}
	return indexCount > 0, nil
}
