'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';

interface Order {
  id: string;
  customer_id: string;
  status: string;
  total_amount: number;
  currency: string;
  items: { product_id: string; quantity: number; unit_price: number }[] | null;
  created_at: string;
}

interface SagaStep {
  step_name: string;
  status: string;
  error: string;
  started_at: string | null;
  completed_at: string | null;
}

interface Saga {
  id: string;
  order_id: string;
  status: string;
  steps: SagaStep[];
  failure_reason: string;
  started_at: string;
  completed_at: string | null;
}

const stepIcons: Record<string, string> = {
  PENDING: '\u23F3',
  EXECUTING: '\uD83D\uDD04',
  COMPLETED: '\u2705',
  FAILED: '\u274C',
  COMPENSATING: '\u21A9\uFE0F',
  COMPENSATED: '\uD83D\uDD19',
};

const statusColors: Record<string, string> = {
  CREATED: 'text-blue-400',
  CONFIRMED: 'text-green-400',
  CANCELLED: 'text-red-400',
  REJECTED: 'text-gray-400',
  PAYMENT_PENDING: 'text-orange-400',
};

export default function OrderDetail() {
  const params = useParams();
  const id = params.id as string;
  const [order, setOrder] = useState<Order | null>(null);
  const [saga, setSaga] = useState<Saga | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const orderRes = await fetch(`http://localhost:8080/api/v1/orders/${id}`);
        setOrder(await orderRes.json());
      } catch (err) {
        console.error('Failed to fetch order:', err);
      }

      try {
        const sagaRes = await fetch(`http://localhost:8080/api/v1/saga/${id}`);
        if (sagaRes.ok) setSaga(await sagaRes.json());
      } catch {}
    };

    fetchData();
    const interval = setInterval(fetchData, 2000);
    return () => clearInterval(interval);
  }, [id]);

  if (!order) return <div className="min-h-screen bg-gray-950 text-white p-8">Loading...</div>;

  return (
    <main className="min-h-screen bg-gray-950 text-white p-8">
      <div className="max-w-3xl mx-auto">
        <a href="/" className="text-gray-400 hover:text-white mb-4 inline-block">&larr; Back to orders</a>

        <div className="bg-gray-900 rounded-xl border border-gray-800 p-6 mb-6">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h1 className="text-xl font-bold">Order {order.id.slice(0, 8)}...</h1>
              <p className="text-gray-400 text-sm mt-1">Customer: {order.customer_id}</p>
            </div>
            <span className={`text-lg font-bold ${statusColors[order.status] || 'text-white'}`}>
              {order.status}
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-400">Total:</span>
              <span className="ml-2 font-mono font-bold">${order.total_amount.toFixed(2)} {order.currency}</span>
            </div>
            <div>
              <span className="text-gray-400">Created:</span>
              <span className="ml-2">{new Date(order.created_at).toLocaleString()}</span>
            </div>
          </div>

          {order.items && order.items.length > 0 && (
            <div className="mt-4 pt-4 border-t border-gray-800">
              <h3 className="text-sm text-gray-400 mb-2">Items</h3>
              {order.items.map((item, i) => (
                <div key={i} className="flex justify-between py-1 text-sm">
                  <span>{item.product_id} x{item.quantity}</span>
                  <span className="font-mono">${(item.unit_price * item.quantity).toFixed(2)}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {saga && (
          <div className="bg-gray-900 rounded-xl border border-gray-800 p-6">
            <h2 className="text-lg font-bold mb-4">Saga Timeline</h2>
            <div className="space-y-3">
              {saga.steps.map((step, i) => (
                <div key={i} className="flex items-center gap-3 p-3 bg-gray-800/50 rounded-lg">
                  <span className="text-xl">{stepIcons[step.status] || '\u23F3'}</span>
                  <div className="flex-1">
                    <div className="font-medium text-sm">{step.step_name.replace(/_/g, ' ')}</div>
                    {step.error && <div className="text-red-400 text-xs mt-1">{step.error}</div>}
                  </div>
                  <span className="text-xs text-gray-400">{step.status}</span>
                </div>
              ))}
            </div>
            {saga.failure_reason && (
              <div className="mt-4 p-3 bg-red-900/20 border border-red-800 rounded-lg text-red-400 text-sm">
                Failure: {saga.failure_reason}
              </div>
            )}
          </div>
        )}
      </div>
    </main>
  );
}
