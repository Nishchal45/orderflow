'use client';

import { useState, useEffect } from 'react';

interface Order {
  id: string;
  customer_id: string;
  status: string;
  total_amount: number;
  currency: string;
  created_at: string;
}

const statusColors: Record<string, string> = {
  CREATED: 'bg-blue-500',
  INVENTORY_RESERVING: 'bg-yellow-500',
  PAYMENT_PENDING: 'bg-orange-500',
  CONFIRMED: 'bg-green-500',
  SHIPPED: 'bg-emerald-600',
  ROLLING_BACK: 'bg-red-400',
  CANCELLED: 'bg-red-600',
  REJECTED: 'bg-gray-500',
};

export default function Home() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchOrders = async () => {
    try {
      const res = await fetch('http://localhost:8080/api/v1/orders');
      const data = await res.json();
      setOrders(data.orders || []);
    } catch (err) {
      console.error('Failed to fetch orders:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrders();
    const interval = setInterval(fetchOrders, 2000);
    return () => clearInterval(interval);
  }, []);

  return (
    <main className="min-h-screen bg-gray-950 text-white p-8">
      <div className="max-w-6xl mx-auto">
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-3xl font-bold">OrderFlow</h1>
            <p className="text-gray-400 mt-1">Real-time order tracking dashboard</p>
          </div>
          <a
            href="/create"
            className="bg-violet-600 hover:bg-violet-700 px-6 py-2 rounded-lg font-medium transition"
          >
            + New Order
          </a>
        </div>

        <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400 text-sm">
                <th className="text-left p-4">Order ID</th>
                <th className="text-left p-4">Customer</th>
                <th className="text-left p-4">Status</th>
                <th className="text-right p-4">Total</th>
                <th className="text-right p-4">Created</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={5} className="text-center p-8 text-gray-500">Loading orders...</td>
                </tr>
              ) : orders.length === 0 ? (
                <tr>
                  <td colSpan={5} className="text-center p-8 text-gray-500">No orders yet. Create one!</td>
                </tr>
              ) : (
                orders.map((order) => (
                  <tr
                    key={order.id}
                    className="border-b border-gray-800/50 hover:bg-gray-800/50 cursor-pointer transition"
                    onClick={() => window.location.href = `/orders/${order.id}`}
                  >
                    <td className="p-4 font-mono text-sm text-gray-300">{order.id.slice(0, 8)}...</td>
                    <td className="p-4">{order.customer_id}</td>
                    <td className="p-4">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${statusColors[order.status] || 'bg-gray-600'}`}>
                        {order.status}
                      </span>
                    </td>
                    <td className="p-4 text-right font-mono">${order.total_amount.toFixed(2)}</td>
                    <td className="p-4 text-right text-gray-400 text-sm">{new Date(order.created_at).toLocaleTimeString()}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <p className="text-gray-600 text-sm mt-4 text-center">Auto-refreshes every 2 seconds</p>
      </div>
    </main>
  );
}
