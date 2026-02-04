#!/usr/bin/env python3
"""
TPC-H and TPC-DS Benchmark Suite for Trino and Spark
Executes queries against both engines and captures performance metrics
"""

import argparse
import json
import csv
import time
import logging
import subprocess
from dataclasses import dataclass, asdict
from typing import List, Dict
from datetime import datetime
import pandas as pd
import matplotlib.pyplot as plt
from pathlib import Path

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@dataclass
class QueryResult:
    """Stores query execution result"""
    query_id: str
    engine: str
    iteration: int
    execution_time_ms: float
    rows_processed: int
    memory_used_mb: float
    cpu_utilization_percent: float
    timestamp: str
    query_text: str = ""


class TrinoClient:
    """Client for executing queries on Trino"""
    
    def __init__(self, host: str = "localhost", port: int = 8080):
        self.host = host
        self.port = port
        self.url = f"http://{host}:{port}"
        logger.info(f"Initialized Trino client: {self.url}")
    
    def execute_query(self, query: str, catalog: str = "hive") -> Dict:
        """Execute query and return results"""
        import urllib.request
        import json as json_lib
        
        try:
            headers = {
                "X-Trino-Client-Info": "benchmark-suite",
                "X-Trino-Session": f"catalog={catalog}",
            }
            
            req = urllib.request.Request(
                f"{self.url}/v1/statement",
                data=query.encode('utf-8'),
                headers=headers,
                method="POST"
            )
            
            with urllib.request.urlopen(req, timeout=300) as response:
                result = json_lib.loads(response.read().decode('utf-8'))
                return result
        except Exception as e:
            logger.error(f"Query execution failed: {e}")
            raise


class SparkClient:
    """Client for executing queries on Spark SQL"""
    
    def __init__(self, host: str = "localhost", port: int = 7077):
        self.host = host
        self.port = port
        logger.info(f"Initialized Spark client: {host}:{port}")
    
    def execute_query(self, query: str) -> Dict:
        """Execute query via Spark API"""
        try:
            # In production, use spark-submit or Livy API
            cmd = [
                "spark-sql",
                "-e", query,
                "--master", f"spark://{self.host}:{self.port}",
                "--conf", "spark.sql.adaptive.enabled=true"
            ]
            
            start = time.time()
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
            elapsed = time.time() - start
            
            if result.returncode != 0:
                logger.error(f"Spark query failed: {result.stderr}")
                raise RuntimeError(result.stderr)
            
            return {
                "output": result.stdout,
                "execution_time_ms": elapsed * 1000,
                "status": "success"
            }
        except Exception as e:
            logger.error(f"Query execution failed: {e}")
            raise


class BenchmarkSuite:
    """Orchestrates benchmark execution"""
    
    # TPC-H Queries (simplified examples)
    TPCH_QUERIES = {
        "tpch-q1": """
            SELECT 
                l_returnflag, 
                l_linestatus, 
                SUM(l_quantity) as sum_qty,
                SUM(l_extendedprice) as sum_base_price,
                SUM(l_extendedprice * (1 - l_discount)) as sum_disc_price,
                SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)) as sum_charge,
                AVG(l_quantity) as avg_qty,
                AVG(l_extendedprice) as avg_price,
                AVG(l_discount) as avg_disc,
                COUNT(*) as count_order
            FROM lineitem
            WHERE l_shipdate <= DATE '1998-12-01'
            GROUP BY l_returnflag, l_linestatus
            ORDER BY l_returnflag, l_linestatus
        """,
        "tpch-q3": """
            SELECT 
                l_orderkey, 
                SUM(l_extendedprice * (1 - l_discount)) as revenue,
                o_orderdate, 
                o_shippriority
            FROM customer c
            JOIN orders o ON c.c_custkey = o.o_custkey
            JOIN lineitem l ON o.o_orderkey = l.l_orderkey
            WHERE c.c_mktsegment = 'BUILDING'
            AND o_orderdate < DATE '1995-03-15'
            AND l_shipdate > DATE '1995-03-15'
            GROUP BY l_orderkey, o_orderdate, o_shippriority
            ORDER BY revenue DESC, o_orderdate
            LIMIT 10
        """,
    }
    
    # TPC-DS Queries
    TPCDS_QUERIES = {
        "tpcds-q1": """
            SELECT customer_id, COUNT(*) as purchase_count
            FROM store_sales
            GROUP BY customer_id
            HAVING COUNT(*) > 10
            ORDER BY purchase_count DESC
        """,
    }
    
    def __init__(self, engine: str, scale: str = "1"):
        self.engine = engine
        self.scale = scale
        self.results: List[QueryResult] = []
        
        if engine == "trino":
            self.client = TrinoClient()
        elif engine == "spark":
            self.client = SparkClient()
        else:
            raise ValueError(f"Unknown engine: {engine}")
    
    def get_query(self, query_id: str) -> str:
        """Retrieve query text"""
        if query_id.startswith("tpch"):
            return self.TPCH_QUERIES.get(query_id, "")
        elif query_id.startswith("tpcds"):
            return self.TPCDS_QUERIES.get(query_id, "")
        return ""
    
    def run_benchmark(self, query_id: str, iterations: int = 3) -> List[QueryResult]:
        """Execute benchmark query multiple times"""
        query = self.get_query(query_id)
        if not query:
            logger.error(f"Query not found: {query_id}")
            return []
        
        logger.info(f"Running {query_id} on {self.engine} ({iterations} iterations)")
        
        for i in range(iterations):
            logger.info(f"Iteration {i+1}/{iterations}")
            
            try:
                start_time = time.time()
                result = self.client.execute_query(query)
                execution_time = (time.time() - start_time) * 1000
                
                query_result = QueryResult(
                    query_id=query_id,
                    engine=self.engine,
                    iteration=i + 1,
                    execution_time_ms=execution_time,
                    rows_processed=result.get("rows", 0),
                    memory_used_mb=result.get("memory_used_mb", 0),
                    cpu_utilization_percent=result.get("cpu_percent", 0),
                    timestamp=datetime.now().isoformat(),
                    query_text=query
                )
                
                self.results.append(query_result)
                logger.info(f"  Time: {execution_time:.2f}ms")
                
            except Exception as e:
                logger.error(f"  Failed: {e}")
                continue
        
        return self.results
    
    def export_json(self, filepath: str):
        """Export results as JSON"""
        data = [asdict(r) for r in self.results]
        with open(filepath, 'w') as f:
            json.dump(data, f, indent=2)
        logger.info(f"Exported {len(data)} results to {filepath}")
    
    def export_csv(self, filepath: str):
        """Export results as CSV"""
        if not self.results:
            return
        
        with open(filepath, 'w', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=asdict(self.results[0]).keys())
            writer.writeheader()
            for result in self.results:
                writer.writerow(asdict(result))
        logger.info(f"Exported {len(self.results)} results to {filepath}")
    
    def generate_report(self, output_dir: str = "./results"):
        """Generate comparison report with charts"""
        if not self.results:
            logger.warning("No results to report")
            return
        
        Path(output_dir).mkdir(exist_ok=True)
        
        df = pd.DataFrame([asdict(r) for r in self.results])
        
        # Generate plots
        fig, axes = plt.subplots(2, 2, figsize=(14, 10))
        
        # Execution time
        axes[0, 0].plot(df['iteration'], df['execution_time_ms'], marker='o')
        axes[0, 0].set_title(f'{self.results[0].engine} - Execution Time')
        axes[0, 0].set_xlabel('Iteration')
        axes[0, 0].set_ylabel('Time (ms)')
        axes[0, 0].grid(True)
        
        # Memory usage
        axes[0, 1].plot(df['iteration'], df['memory_used_mb'], marker='s', color='orange')
        axes[0, 1].set_title(f'{self.results[0].engine} - Memory Usage')
        axes[0, 1].set_xlabel('Iteration')
        axes[0, 1].set_ylabel('Memory (MB)')
        axes[0, 1].grid(True)
        
        # CPU utilization
        axes[1, 0].plot(df['iteration'], df['cpu_utilization_percent'], marker='^', color='green')
        axes[1, 0].set_title(f'{self.results[0].engine} - CPU Utilization')
        axes[1, 0].set_xlabel('Iteration')
        axes[1, 0].set_ylabel('CPU (%)')
        axes[1, 0].grid(True)
        
        # Statistics table
        stats = df[['execution_time_ms', 'memory_used_mb', 'cpu_utilization_percent']].describe()
        axes[1, 1].axis('off')
        axes[1, 1].text(0.1, 0.9, stats.to_string(), 
                       transform=axes[1, 1].transAxes, fontfamily='monospace')
        
        plt.tight_layout()
        report_path = f"{output_dir}/benchmark_report_{self.engine}.png"
        plt.savefig(report_path, dpi=150)
        logger.info(f"Saved report to {report_path}")


def main():
    parser = argparse.ArgumentParser(description="TPC Benchmark Suite")
    parser.add_argument("--engine", required=True, choices=["trino", "spark"],
                       help="Query engine to benchmark")
    parser.add_argument("--query", default="tpch-q1",
                       help="Query ID to run (e.g., tpch-q1, tpcds-q1)")
    parser.add_argument("--scale", default="1",
                       help="TPC scale factor")
    parser.add_argument("--iterations", type=int, default=3,
                       help="Number of iterations")
    parser.add_argument("--output-format", choices=["json", "csv", "both"], default="json",
                       help="Output format")
    parser.add_argument("--output-dir", default="./results",
                       help="Output directory for results")
    
    args = parser.parse_args()
    
    # Run benchmark
    suite = BenchmarkSuite(args.engine, args.scale)
    suite.run_benchmark(args.query, args.iterations)
    
    # Export results
    output_dir = args.output_dir
    Path(output_dir).mkdir(exist_ok=True)
    
    if args.output_format in ["json", "both"]:
        suite.export_json(f"{output_dir}/results_{args.engine}.json")
    
    if args.output_format in ["csv", "both"]:
        suite.export_csv(f"{output_dir}/results_{args.engine}.csv")
    
    # Generate report
    suite.generate_report(output_dir)
    
    # Print summary
    if suite.results:
        avg_time = sum(r.execution_time_ms for r in suite.results) / len(suite.results)
        logger.info(f"\n{'='*50}")
        logger.info(f"BENCHMARK SUMMARY")
        logger.info(f"Engine: {args.engine}")
        logger.info(f"Query: {args.query}")
        logger.info(f"Iterations: {args.iterations}")
        logger.info(f"Average execution time: {avg_time:.2f}ms")
        logger.info(f"{'='*50}\n")


if __name__ == "__main__":
    main()
