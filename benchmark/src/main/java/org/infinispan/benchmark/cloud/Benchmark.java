package org.infinispan.benchmark.cloud;

import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.infinispan.client.hotrod.RemoteCache;
import org.infinispan.client.hotrod.RemoteCacheManager;
import org.infinispan.client.hotrod.configuration.ClientIntelligence;
import org.infinispan.client.hotrod.configuration.ConfigurationBuilder;
import org.openjdk.jmh.annotations.Mode;
import org.openjdk.jmh.annotations.Param;
import org.openjdk.jmh.annotations.Scope;
import org.openjdk.jmh.annotations.Setup;
import org.openjdk.jmh.annotations.State;
import org.openjdk.jmh.annotations.TearDown;
import org.openjdk.jmh.runner.Runner;
import org.openjdk.jmh.runner.RunnerException;
import org.openjdk.jmh.runner.options.Options;
import org.openjdk.jmh.runner.options.OptionsBuilder;

public class Benchmark {

   private static final int MEASUREMENT_ITERATIONS_COUNT = 31;
   private static final int WARMUP_ITERATIONS_COUNT = 5;

   public static void main(String[] args) throws RunnerException {

      Options opt = new OptionsBuilder()
            .include(Benchmark.class.getName() + ".*")
            .mode(Mode.AverageTime)
            .timeUnit(TimeUnit.MILLISECONDS)
            .warmupIterations(WARMUP_ITERATIONS_COUNT)
            .measurementIterations(MEASUREMENT_ITERATIONS_COUNT)
            .threads(1)
            .forks(1)
            .shouldFailOnError(true)
            .shouldDoGC(true)
            .build();

      new Runner(opt).run();
   }

   @State(Scope.Thread)
   public static class BenchmarkState {

      @Param({
            "true",
            "false"
      })
      public boolean useSmartBalancing;

      RemoteCacheManager cacheManager;
      RemoteCache<String, String> cache;

      @Setup
      public void setup() throws Exception {
         ConfigurationBuilder builder = new ConfigurationBuilder();

         if(useSmartBalancing) {
            builder.addServer()
                  .host("104.155.17.202").port(11222)
                  .host("104.199.78.28").port(11222)
                  .host("130.211.49.127").port(11222)
                  .addressMapping(CloudAddressMapper.class);
         } else {
            builder.clientIntelligence(ClientIntelligence.BASIC);
            builder.addServer()
                  .host("104.155.17.202").port(11222)
                  .host("104.199.78.28").port(11222)
                  .host("130.211.49.127").port(11222);
         }
         cacheManager = new RemoteCacheManager(builder.build());
         cache = cacheManager.getCache();
         cache.clear();
         Thread.sleep(500);
      }

      @TearDown
      public void tearDown() throws Exception {
         cache.stop();
         cacheManager.stop();
      }

      @org.openjdk.jmh.annotations.Benchmark
      public void perform1000Puts() throws Exception {
         for (int i = 0; i < 1_000; ++i) {
            String key = UUID.randomUUID().toString();
            cache.put(key, key);
         }
      }

      @org.openjdk.jmh.annotations.Benchmark
      public void perform1000PutsAndGets() throws Exception {
         for (int i = 0; i < 1_500; ++i) {
            String key = UUID.randomUUID().toString();
            cache.put(key, key);
            cache.get(key);
         }
      }

   }

}
