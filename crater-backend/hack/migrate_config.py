import yaml
import copy

def migrate_config(old_config):
    new_config = {}
    
    # Directly copied fields
    direct_fields = [
        'enableLeaderElection', 'leaderElectionID', 'prometheusAPI',
        'host', 'serverAddr', 'metricsAddr', 'probeAddr'
    ]
    for field in direct_fields:
        if field in old_config:
            new_config[field] = old_config[field]
    
    # Postgres (direct copy)
    if 'postgres' in old_config:
        new_config['postgres'] = copy.deepcopy(old_config['postgres'])
    
    # Auth section
    new_config['auth'] = {}
    if 'act' in old_config and 'auth' in old_config['act']:
        act_auth = old_config['act']['auth']
        if 'accessTokenSecret' in act_auth:
            new_config['auth']['accessTokenSecret'] = act_auth['accessTokenSecret']
        if 'refreshTokenSecret' in act_auth:
            new_config['auth']['refreshTokenSecret'] = act_auth['refreshTokenSecret']
    
    # Storage section
    new_config['storage'] = {
        'prefix': {}
    }
    if 'userSpacePrefix' in old_config:
        new_config['storage']['prefix']['user'] = old_config['userSpacePrefix']
    if 'accountSpacePrefix' in old_config:
        new_config['storage']['prefix']['account'] = old_config['accountSpacePrefix']
    if 'publicSpacePrefix' in old_config:
        new_config['storage']['prefix']['public'] = old_config['publicSpacePrefix']
    
    if 'workspace' in old_config:
        if 'rwxpvcName' in old_config['workspace']:
            new_config['storage']['rwxpvcName'] = old_config['workspace']['rwxpvcName']
        if 'roxpvcName' in old_config['workspace']:
            new_config['storage']['roxpvcName'] = old_config['workspace']['roxpvcName']
    
    # Workspace
    new_config['workspace'] = {}
    if 'workspace' in old_config:
        workspace = old_config['workspace']
        if 'namespace' in workspace:
            new_config['workspace']['namespace'] = workspace['namespace']
        if 'imageNameSpace' in workspace:
            new_config['workspace']['imageNameSpace'] = workspace['imageNameSpace']
    
    # Secrets
    new_config['secrets'] = {}
    if 'tlsSecretName' in old_config:
        new_config['secrets']['tlsSecretName'] = old_config['tlsSecretName']
    if 'tlsForwardSecretName' in old_config:
        new_config['secrets']['tlsForwardSecretName'] = old_config['tlsForwardSecretName']
    if 'imagePullSecretName' in old_config:
        new_config['secrets']['imagePullSecretName'] = old_config['imagePullSecretName']
    
    # ImageRegistry
    if 'act' in old_config and 'image' in old_config['act']:
        act_image = old_config['act']['image']
        new_config['imageRegistry'] = {
            'server': act_image.get('registryServer', ''),
            'user': act_image.get('registryUser', ''),
            'password': act_image.get('registryPass', ''),
            'project': act_image.get('registryProject', ''),
            'admin': act_image.get('registryAdmin', ''),
            'adminPassword': act_image.get('registryAdminPass', '')
        }
    
    # ImageBuildTools (formerly DindArgs)
    if 'dindArgs' in old_config:
        dind_args = old_config['dindArgs']
        new_config['imageBuildTools'] = {
            'buildxImage': dind_args.get('buildxImage', ''),
            'nerdctlImage': dind_args.get('nerdctlImage', ''),
            'envdImage': dind_args.get('envdImage', '')
        }
    
    # SMTP
    if 'act' in old_config and 'smtp' in old_config['act']:
        new_config['smtp'] = copy.deepcopy(old_config['act']['smtp'])
    
    # RaidsLab (formerly ACT)
    new_config['raidsLab'] = {
        'ldap': {},
        'openAPI': {}
    }
    
    if 'act' in old_config:
        # StrictRegisterMode -> Enable
        if 'strictRegisterMode' in old_config['act']:
            new_config['raidsLab']['enable'] = old_config['act']['strictRegisterMode']
        
        # UIDServerURL
        if 'uidServerURL' in old_config['act']:
            new_config['raidsLab']['uidServerURL'] = old_config['act']['uidServerURL']
        
        # OpenAPI
        if 'openAPI' in old_config['act']:
            new_config['raidsLab']['openAPI'] = copy.deepcopy(old_config['act']['openAPI'])
        
        # LDAP (from ACT.auth)
        if 'auth' in old_config['act']:
            act_auth = old_config['act']['auth']
            ldap_mapping = {
                'userName': 'userName',
                'password': 'password',
                'address': 'address',
                'searchDN': 'searchDN'
            }
            for old_key, new_key in ldap_mapping.items():
                if old_key in act_auth:
                    new_config['raidsLab']['ldap'][new_key] = act_auth[old_key]
    
    # SchedulerPlugins (direct copy)
    if 'schedulerPlugins' in old_config:
        new_config['schedulerPlugins'] = copy.deepcopy(old_config['schedulerPlugins'])
    
    return new_config

if __name__ == "__main__":
    # Read old and new from cmd
    import sys
    if len(sys.argv) != 3:
        print("Usage: python migrate_config.py <old_config_path> <new_config_path>")
        sys.exit(1)

    # Read old config
    with open(sys.argv[1], 'r') as f:
        old_config = yaml.safe_load(f)
    
    # Migrate config
    new_config = migrate_config(old_config)
    
    # Write new config
    with open(sys.argv[2], 'w') as f:
        yaml.dump(new_config, f, sort_keys=False, default_flow_style=False)