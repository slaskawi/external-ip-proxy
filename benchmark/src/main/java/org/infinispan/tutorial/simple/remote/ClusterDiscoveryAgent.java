package org.infinispan.tutorial.simple.remote;

import java.io.IOException;
import java.net.URL;
import java.util.List;
import java.util.Map;

import org.yaml.snakeyaml.Yaml;

public class ClusterDiscoveryAgent {

   private final String exposedMappingServiceUrl;

   public ClusterDiscoveryAgent(String discoveryServiceAddress) {
      exposedMappingServiceUrl = discoveryServiceAddress;
   }

   public DiscoveryInfo discover() throws IOException {
      Yaml yaml = new Yaml();
      Map<String, Map> configuration = (Map<String, Map>) yaml.loadAll(new URL(exposedMappingServiceUrl).openStream()).iterator().next();
      Map<String, Map> runtimeConfiguration = (Map<String, Map>) configuration.get("runtime-configuration");
      List<String> mappings = (List<String>) runtimeConfiguration.get("external-mapping");
      return new DiscoveryInfo(mappings);
   }



}
