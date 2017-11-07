# vRA Automation through REST endpoint

### Requirements

- ruby 2.2+
- sudo gem install rest-client
- sudo gem install nokogiri
- The following environment variables are required:

`VRA_USER` , `VRA_PASSWORD` , `VRA_TENANT` , `VRA_ENDPOINT` , `VRA_BG`


### Request Blueprint

This will stand up and existing blueprint

`ruby ./vmRequest.rb [blueprint_name]`

This command should invoke a vRA blueprint and monitor
it's progress until it completes.

## Destroy Deployment

This will remove all resources for a given deployment ID

`ruby ./vmDestroy.rb [deployment_id]`