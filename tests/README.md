# Testing

We don't use some library of package for testing. It is just pure python code that runs command line commands and reads response.

Tests are all python scripts in this folder which doesn't start from _

## Usage

Execute all tests:

```
python _tester.py
```

Execute custom test:

```
python _tester.py managenodes
```

### Example of success test

```
$ python _tester.py startnode.py
===================Start/Stop node======================
        ----------------Try to start without blockchain
===================Init blockchain======================
        ----------------Create first address
        ----------------Create blockchain
===================Start node ======================
        ----------------Start normal
        ----------------Check node state
        ----------------Start node again. should not be allowed
===================Stop node ======================
        ----------------Check node state
        ----------------Stop node
        ----------------Stop node again
PASS ===
```