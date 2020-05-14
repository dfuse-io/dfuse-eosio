dfuse Connector to SQL
======================

Design:

* Start process
  * hard-configured for `simpleassets`
  * read database DSN flag
* Get the ABI at HEAD (figure out the latest irreversible block processed by FluxDB)
  * Retrieve the `lib_num` in the response
  * Get the ABI at `lib_num`
* Open SQL database
* Loop all the 8-10 tables of `simpleassets`:
  * Get create table statement, by calling a (hard-coded) `abi_to_sql` (with
    potential overrides from a declarative file eventually)
  * Send CREATE TABLE statements
* Fetch a snapshot of all these tables before sync'ing:
  * For all 8-10 tables:
    * Get all the scopes at `lib_num`, with json=false
    * ABI decode incoming rows
    * Map to a `.csv`
    * Stash into a BatcherInserter for each table
  * Open transaction
  * For each table:
    * Flush each table with INSERT INTO
  * Write a _marker_ (written block ref) in the DB to know where we
    were at (to read and pickup in Pipeline mode)
  * Commit transaction
* Start Pipeline:
  * Read the _marker_
  * Kickstart a Joining source + Live + whatever's needed
  * ProcessBlock:
    * Watch out ABI changes. What if it changes? oh oh!
    * Loop through all transactions, actions, identify the tables that match this contract
      * Each change to a table is added to a its corresponding BatchInserter
    * Open SQL trx
    * Flush all BatchInserters
    * Write _marker_
    * Commit trx
