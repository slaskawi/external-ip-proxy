package org.infinispan.tutorial.simple.remote;

import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.infinispan.client.hotrod.RemoteCache;
import org.infinispan.client.hotrod.RemoteCacheManager;
import org.infinispan.client.hotrod.configuration.ClientIntelligence;
import org.infinispan.client.hotrod.configuration.ConfigurationBuilder;
import org.openjdk.jmh.annotations.Benchmark;
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

public class InfinispanRemote {

   private static final int MEASUREMENT_ITERATIONS_COUNT = 31;
   private static final int WARMUP_ITERATIONS_COUNT = 0;

   public static void main(String[] args) throws RunnerException {

      Options opt = new OptionsBuilder()
            .include(InfinispanRemote.class.getName() + ".*")
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
      public boolean useLoadBalancerPerPod;

      RemoteCacheManager cacheManager;
      RemoteCache<String, String> cache;

      @Setup
      public void setup() throws Exception {
         ConfigurationBuilder builder = new ConfigurationBuilder();
         builder.clientIntelligence(ClientIntelligence.BASIC);
         if(useLoadBalancerPerPod) {
            builder.addServer().host("172.29.0.2").port(11222);
         } else {
            builder.addServer().host("172.17.0.3").port(11222);
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

      @Benchmark
      public void perform1000Puts() throws Exception {
         for (int i = 0; i < 1_000; ++i) {
            String key = UUID.randomUUID().toString();
            cache.put(key, key);
         }
      }

      @Benchmark
      public void perform1000PutsAndGets() throws Exception {
         for (int i = 0; i < 1_000; ++i) {
            String key = UUID.randomUUID().toString();
            cache.put(key, key);
            cache.get(key);
         }
      }

   }

}
