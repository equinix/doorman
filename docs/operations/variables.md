# Environment Variables

Required variables:

1. DOORMAN_API_HOST - [FQDN](https://en.wikipedia.org/wiki/Fully_qualified_domain_name) of the Equinix Metal API that will be used to get user specific data.
   For example: "https://api.equinix.com".
   
1. FACILITY - Equinix facility code where this software will be deployed.  
   For example: "ny5", "sv15".

1. EQUINIX_ENV - production or testing

1. EQUINIX_VERSION - git hash that this build is based on.  
   This it typically filled by the CI/CD system.
      
1. PROMETHUES_SERVER_PORT - Port that built in Promethues server should listen on.  
   Default value is ":9090".