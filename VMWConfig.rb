require 'VMW'
require 'VMWPayload'

missing_env = []
ENV["VRA_ENDPOINT"] ? VMW::API::baseURI = ENV["VRA_ENDPOINT"] : missing_env << "VRA_ENDPOINT"
ENV["VRA_USER"] ? VMW::API::userName = ENV["VRA_USER"] : missing_env << "VRA_USER"
ENV["VRA_PASSWORD"] ? VMW::API::password = ENV["VRA_PASSWORD"] : missing_env << "VRA_PASSWORD"
ENV["VRA_TENANT"] ? VMW::API::tenant = ENV["VRA_TENANT"] : missing_env << "VRA_TENANT"

unless missing_env.empty?
  puts "Missing the following environment variables: %s" % missing_env.to_s
  exit 1
end

VMW::API::debug = false

# Set this to always require authentication on every http request
#VMW::API::autoEnableAuth

# Set this to false if want to ignore problems with self-signed certs.
VMW::API::sslVerify = false

VMW::Payload.basePath = './payloads'

# hack to deal with self-signed certificates
module RestClient
  class Request
    def self.execute(args, & block)
      unless VMW::API::sslVerify
        args[:verify_ssl] = false
      end
      new(args).execute(& block)
    end
  end
end
