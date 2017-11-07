libdir = File.dirname(__FILE__)
$LOAD_PATH.unshift(libdir) unless $LOAD_PATH.include?(libdir)

require 'VMWConfig'

if ARGV[0]
    resource_id = ARGV[0]
else
    puts "Parent Resource ID required as first argument"
end

url = VMW::API::URI('/catalog-service/api/consumer/resources')

begin
    response = VMW::API::sign {
        RestClient.get url, :content_type => :json, :params => {:$filter => "id eq '%s'" % resource_id}
    }
    payload = VMW::Payload.from_json(response)
    content = payload.doc["content"].first
    resource_name = content["name"]
    puts "Found resource with name: %s" % content["name"]

    # Retrieve available actions for the deploymet and look for the destroy
    url = VMW::API::URI("/catalog-service/api/consumer/resources/%s/actions" % resource_id)
    response = VMW::API::sign {
        RestClient.get url, :content_type => :json
    }
    payload = VMW::Payload.from_json(response)
    action_id = nil
    payload.doc["content"].each do  |action|
        if action["bindingId"] == "composition.resource.action.deployment.destroy"
            action_id = action["id"]
        end
    end
    if action_id
        # Get template for destroying deployment
        url = VMW::API::URI("/catalog-service/api/consumer/resources/%s/actions/%s/requests/template" % [resource_id, action_id])
        response = VMW::API::sign {
            RestClient.get url, :content_type => :json
        }
        payload = VMW::Payload.from_json(response)
        template = payload.doc

        # Destroy using template
        url = VMW::API::URI("/catalog-service/api/consumer/resources/%s/actions/%s/requests" % [resource_id, action_id])
        response = VMW::API::sign {
            RestClient.post url, template.to_json, :content_type => :json
        }
        if response.code == 201
            puts "Successfully scheduled the cleanup of deployment: %s, id: %s" % [resource_name, resource_id]
        else
            puts "Error requesting the cleanup of deployment: %s, id: %s" % [resource_name, resource_id]
        end
    else
        puts "The following deployment does not have a destroy entitlement: %s" % resource_id
        exit 1
    end
    
rescue RestClient::Exception => e
    print "Got exception with status: %d\n" % e.response.code
    print "%s\n" % e.response
end


exit 0