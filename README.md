# fitest
Take home tech test for Face IT

# To Do

* Consider defensive coding in the store or user package when accepting UUIDs. Could use parseBytes to ensure validity (test current behaviour first!)
* Document toolchain
    * go 1.18
    * Make
    * staticcheck
    * docker
* Document CI Process
* Document testing locally
* Document the proto file for the rpc
* Document the decision to just use grpc error codes for the sake of simplicity
* Document the decision to hash the password. 
* Document the decision not to share the hashed password
* Document the decision not to verify the old password on update (we are not doing authn/z)
* Document the decision to make email and nickname unique and not updatable
* Set a request timeout