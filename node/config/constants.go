package config

// ==========================================================
// this can be altered to experiment with blockchain

// this defines how strong miming is needed. 16 is simple mining less 5 sec in simple desktop
// 24 will need 30 seconds in average
const TargetBits = 16
const TargetBits_2 = 24

// Max and Min number of transactions per block
// If number of block in a chain is less this umber then it is a minimum. if more then
// this number is  a minimum unmber of TX
const MaxMinNumberTransactionInBlock = 1000

// Max number of TX per block
const MaxNumberTransactionInBlock = 10000

// ==========================================================
//No need to change this

// File names
const PidFileName = "server.pid"

// other internal constant
const Daemonprocesscommandline = "daemonnode"

// ==========================================================
// Testing mode constants
// we need this for testing purposes. can be set to 0 on production system
const MinimumBlockBuildingTime = 3 // seconds
