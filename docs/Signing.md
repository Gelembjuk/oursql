# Signing of transactions in OurSQL

Each SQL update is a blockchain transaction and it must be signed. There are 2 supported ways to sign a transaction.

1. Signing by a node. If a node config has the option "ProxyKey" set (or sommand line argument -dbproxyaddr), than a node will sign itself all transactions coming from through DB proxy (SQL updates from a client).
1. Signing by a MySQL client. It can be used in case if your node is just a DB proxy and many clients connect to it (meeaning different users with different keys). In this case, SQL updates from a mysql client will require 2 steps to execute.
    * First step - send SQL query and add your public key inside a comment of special format. DB proxy returns a record which contains a data to sign (string to sign)
    * Second step - sign a string with a private key corresponding to public key posted on the first step and do new SQL request where a signature is included as part of a comment of special format.

The first mode is simple for a client. Nothing special. But in this mode every user must have own node. 

If you need one node for multiple users, you need to use the second way.

## Signature type

OurSQL uses prime256v1 ECDSA signature. It can be generated with openssl

```
openssl ecparam -name prime256v1 -genkey -noout -out prime256v1-key.pem
openssl ec -in prime256v1-key.pem -text -noout
```

## Example of queries in a second mode

Query

```
CREATE TABLE users (id int auto_increment primary key, name varchar(50))/* PUBKEY:74d8f21f92018ad892c418e8ab94d6b334ba01a6e4c48e15943db1945d24dea53f83d54d79f0c2b5e5123acd63483222e39d6544917f1b604d1aec814a11b1f9; */

#Response will be s string to sign

156652624d27e17074d8f21f92018ad892c418e8ab94d6b334ba01a6e4c48e15943db1945d24dea53f83d54d79f0c2b5e5123acd63483222e39d6544917f1b604d1aec814a11b1f975736572733a2a435245415445205441424c452075736572732028696420696e74206175746f5f696e6372656d656e74207072696d617279206b65792c206e616d652076617263686172283530292944524f50205441424c45207573657273

# Repeat query with signed string
CREATE TABLE users (id int auto_increment primary key, name varchar(50))/* DATA:75ff810301010b5472616e73616374696f6e01ff8200010801024944010a00010454696d6501040001095369676e6174757265010a00010842795075624b6579010a00010356696e01ff86000104566f757401ff8a00010a53514c436f6d6d616e6401ff8c00010953514c426173655458010a0000002bff850201011c5b5d737472756374757265732e545843757272656e6379496e70757401ff860001ff8400002fff830301010f545843757272656e6379496e70757401ff84000102010454786964010a000104566f757401040000002dff890201011e5b5d737472756374757265732e54584375727272656e63794f757470757401ff8a0001ff88000038ff870301011154584375727272656e63794f757470757401ff88000102010556616c7565010800010a5075624b657948617368010a00000057ff8b0301010953514c55706461746501ff8c000104010b5265666572656e63654944010a0001055175657279010a00010d526f6c6c6261636b5175657279010a00010f507265765472616e73616374696f6e010a000000ffb6ff8202f82acca4c49a4fc2e0024074d8f21f92018ad892c418e8ab94d6b334ba01a6e4c48e15943db1945d24dea53f83d54d79f0c2b5e5123acd63483222e39d6544917f1b604d1aec814a11b1f903010775736572733a2a0148435245415445205441424c452075736572732028696420696e74206175746f5f696e6372656d656e74207072696d617279206b65792c206e616d6520766172636861722835302929011044524f50205441424c452075736572730000; SIGN:304502200e0679da9db415bd6b2f9ebfbd1714efe8eaad8faae457f3c0b88bc4fc8c4027022100ff89dfffda41532d5c0cf06412b81e4c68e22febd1486e1b86467d44a51792f6;*/

# This query contains 
SIGN - which is the string to sign returned from previous call signed with private key
DATA - this is the data of transaction encoded in bytes. It is received from the first request together with string to sign and must be posted back on second request
```

## Sample code with PHP to execute signed SQL update

```
<?php

require '../../src/vendor/autoload.php';

$servername = "localhost:8766";
$username = "blockchain";
$password = "blockchain";
$dbname = "BC";

$keys = '/path/to/wallet/wallet.dat'; // wallet is same format as OurSQL stores
$wallets = json_decode(file_get_contents($keys),true);

global $conn, $wallet, $privatekey;

$wallet = $wallets['Wallets'][0]['PubKey'];
$privatekey = $wallets['Wallets'][0]['PrivateKey'];

// Create MySQL connection
$conn = new mysqli($servername, $username, $password, $dbname);

if ($conn->connect_error) {
    die("Connection failed: " . $conn->connect_error);
} 

$result = executeSignedQuery("CREATE TABLE users (id int auto_increment primary key, name varchar(50))");

if ($result != '') {
    echo $result."\n";
    exit;
}

$result = executeSignedQuery("INSERT INTO users SET name='user7'");


function executeSignedQuery($sql)
{
    global $conn,$wallet,$privatekey;
    
    $prkey = openssl_pkey_get_private($privatekey);

    if ($prkey == false) {
        return "Failed to open key ".openssl_error_string();
    }

    $result = $conn->query($sql."/* PUBKEY:$wallet; */");

    if (!$result) {
        return "Last error: ".$conn->error." (".$conn->errno.")";
    } 
    
    if ($result->num_rows == 0) {
        return "No signatre info returned";
    }

    $data = [];
    // output data of each row
    // We executed update SQL query but result will be rows with data to use for signature
    while($row = $result->fetch_assoc()) {
        $data[$row['Key']] = $row['Value'];
    }
    echo $data['StringToSign']."\n\n";
    $stringtosign = pack("H*", $data['StringToSign']);
    
    if(!openssl_sign($stringtosign, $signature, $prkey, OPENSSL_ALGO_SHA1)){
        return "Failed to sign data: ".openssl_error_string();
    }
    
    openssl_free_key($prkey);
    // pack signature to OurSQL formate
    $signature = implode(unpack("H*", $signature));
    
    $finalsql = $sql."/* DATA:".$data['Transaction']."; SIGN:".$signature.";*/";

    $result = $conn->query($finalsql);

    if (!$result) {
        return "Last error: ".$conn->error." (".$conn->errno.")";
    }
    
    return '';
}

$sql = "SELECT * FROM users";
$result = $conn->query($sql);

if ($result->num_rows > 0) {
    while($row = $result->fetch_assoc()) {
        echo "id: " . $row["id"]. " - Name: " . $row["name"]. " " . $row["email"]. "\n";
    }
} else {
    echo "0 results";
}

```


