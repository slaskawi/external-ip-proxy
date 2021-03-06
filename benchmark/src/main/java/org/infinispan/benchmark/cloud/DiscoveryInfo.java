package org.infinispan.benchmark.cloud;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class DiscoveryInfo {

   private Map<String, String> addresses = new HashMap<>();

   public DiscoveryInfo(List<String> mappings) {
      for (String mapping : mappings) {
         String[] separatedAddresses = mapping.split(":");
         addresses.put(separatedAddresses[0], separatedAddresses[1]);
      }
   }

   public Map<String, String> getAddresses() {
      return Collections.unmodifiableMap(addresses);
   }

   public String getExternalAddress(String internalAddress) {
      return addresses.get(internalAddress);
   }

   @Override
   public String toString() {
      return "DiscoveryInfo{" +
            "addresses=" + addresses +
            '}';
   }
}
